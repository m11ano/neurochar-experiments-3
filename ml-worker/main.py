import asyncio
import contextlib
import gc
import logging
import os
import time
import uuid
from dataclasses import dataclass
from typing import Optional, Tuple

import fitz
import numpy as np
import easyocr

from fastapi import FastAPI, File, UploadFile, HTTPException, Request
from fastapi.responses import JSONResponse

import grpc
import ocr_pb2
import ocr_pb2_grpc


APP_TIMEOUT_SECONDS = 5 * 60
MAX_PDF_BYTES = 50 * 1024 * 1024
DPI = int(os.getenv("OCR_DPI", 150))
OCR_CONCURRENCY = int(os.getenv("OCR_CONCURRENCY", 1))

EASYOCR_MODEL_DIR = "/models/easyocr"

GRPC_HOST = "0.0.0.0"
GRPC_PORT = 50051

PING_TIMEOUT_S = 30

DEFAULT_DEVICE = os.getenv("OCR_DEVICE", "auto").strip().lower()

app = FastAPI(title="PDF OCR Service (EasyOCR single-queue, GPU if available)")

OCR_SEMAPHORE = asyncio.Semaphore(OCR_CONCURRENCY)

logger = logging.getLogger("pdf_ocr")
if not logger.handlers:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s | %(levelname)s | %(name)s | %(message)s",
    )


@dataclass(frozen=True)
class OcrOptions:
    dpi: int = DPI
    languages: Tuple[str, ...] = ("ru", "en")
    detail: int = 0
    paragraph: bool = True


def _page_to_numpy_rgb(page: fitz.Page, dpi: int) -> np.ndarray:
    mat = fitz.Matrix(dpi / 72.0, dpi / 72.0)
    pix = page.get_pixmap(matrix=mat, alpha=False)
    try:
        return np.frombuffer(pix.samples, dtype=np.uint8).reshape(pix.height, pix.width, 3).copy()
    finally:
        del pix


def _normalize_easyocr_output(result) -> str:
    if not result:
        return ""

    if isinstance(result[0], str):
        return "\n".join(s.strip() for s in result if s and s.strip()).strip()

    texts = []
    for item in result:
        if len(item) >= 2 and item[1]:
            texts.append(str(item[1]).strip())
    return "\n".join(t for t in texts if t).strip()


READER_CPU: Optional[easyocr.Reader] = None
READER_GPU: Optional[easyocr.Reader] = None
GPU_AVAILABLE: bool = False


def _parse_device(value: Optional[str]) -> str:
    if not value:
        return "auto"
    v = value.strip().lower()
    if v in ("auto",):
        return "auto"
    if v in ("gpu", "cuda"):
        return "gpu"
    if v in ("cpu",):
        return "cpu"
    return "auto"


def _detect_gpu_available() -> bool:
    try:
        import torch
        return bool(torch.cuda.is_available())
    except Exception:
        return False


def _resolve_effective_device(requested: str) -> str:
    requested = _parse_device(requested)
    if requested == "auto":
        return "gpu" if GPU_AVAILABLE else "cpu"
    if requested == "gpu" and not GPU_AVAILABLE:
        return "cpu"
    return requested


def _clear_accelerator_cache(use_gpu: bool) -> None:
    gc.collect()
    if use_gpu:
        try:
            import torch
            if torch.cuda.is_available():
                torch.cuda.empty_cache()
        except Exception:
            pass


def _get_reader(requested_device: str, languages: Tuple[str, ...]) -> tuple[easyocr.Reader, bool]:
    global READER_CPU, READER_GPU

    effective = _resolve_effective_device(requested_device)

    if effective == "gpu":
        if READER_GPU is None:
            logger.info("init EasyOCR GPU reader | languages=%s | model_dir=%s", list(languages), EASYOCR_MODEL_DIR)
            READER_GPU = easyocr.Reader(
                list(languages),
                gpu=True,
                model_storage_directory=EASYOCR_MODEL_DIR,
            )
            warm = np.zeros((10, 10, 3), dtype=np.uint8)
            _ = READER_GPU.readtext(warm, detail=0, paragraph=True)
        return READER_GPU, True

    if READER_CPU is None:
        logger.info("init EasyOCR CPU reader | languages=%s | model_dir=%s", list(languages), EASYOCR_MODEL_DIR)
        READER_CPU = easyocr.Reader(
            list(languages),
            gpu=False,
            model_storage_directory=EASYOCR_MODEL_DIR,
        )
        warm = np.zeros((10, 10, 3), dtype=np.uint8)
        _ = READER_CPU.readtext(warm, detail=0, paragraph=True)
    return READER_CPU, False


@app.on_event("startup")
def _startup():
    global GPU_AVAILABLE
    GPU_AVAILABLE = _detect_gpu_available()

    logger.info("startup: gpu_available=%s | default_device=%s", GPU_AVAILABLE, DEFAULT_DEVICE)

    opts = OcrOptions()
    reader, use_gpu = _get_reader(DEFAULT_DEVICE, opts.languages)
    _ = reader
    logger.info("startup: EasyOCR ready | mode=%s", "GPU(cuda)" if use_gpu else "CPU")


def process_pdf_bytes_ocr_easyocr_always_images(
    pdf_bytes: bytes,
    opts: Optional[OcrOptions] = None,
    *,
    request_id: str = "-",
    device: str = "auto",
) -> dict:
    opts = opts or OcrOptions()

    reader, use_gpu = _get_reader(device, opts.languages)

    try:
        doc = fitz.open(stream=pdf_bytes, filetype="pdf")
    except Exception as e:
        raise ValueError(f"Не удалось открыть PDF: {e}")

    pages: list[dict] = []
    text_parts: list[str] = []

    try:
        total_pages = len(doc)

        for i in range(total_pages):
            page_num = i + 1
            page = None
            img_np = None
            raw = None

            try:
                page = doc.load_page(i)

                logger.info("req=%s | page=%d/%d | render_start", request_id, page_num, total_pages)
                img_np = _page_to_numpy_rgb(page, dpi=opts.dpi)
                logger.info(
                    "req=%s | page=%d/%d | render_done | shape=%s",
                    request_id,
                    page_num,
                    total_pages,
                    tuple(img_np.shape),
                )

                logger.info("req=%s | page=%d/%d | readtext_start", request_id, page_num, total_pages)
                raw = reader.readtext(
                    img_np,
                    detail=opts.detail,
                    paragraph=opts.paragraph,
                )
                logger.info("req=%s | page=%d/%d | readtext_done", request_id, page_num, total_pages)

                text = _normalize_easyocr_output(raw)

                pages.append({
                    "page": page_num,
                    "text": text,
                })
                text_parts.append(f"--- Page {page_num} ---\n{text}".rstrip())

            finally:
                del raw
                del img_np
                del page
                _clear_accelerator_cache(use_gpu)

    finally:
        doc.close()
        _clear_accelerator_cache(use_gpu)

    full_text = "\n\n".join(text_parts).strip()

    return {
        "engine": "easyocr",
        "gpu": bool(use_gpu),
        "device": "cuda" if use_gpu else "cpu",
        "dpi": opts.dpi,
        "languages": list(opts.languages),
        "pages": pages,
        "text": full_text,
        "request_id": request_id,
    }


async def process_pdf_with_queue_and_timeout(
    pdf_bytes: bytes,
    opts: Optional[OcrOptions] = None,
    *,
    request_id: str,
    device: str = "auto",
) -> dict:
    opts = opts or OcrOptions()

    async def _do():
        t_wait_start = time.perf_counter()
        async with OCR_SEMAPHORE:
            waited_s = time.perf_counter() - t_wait_start

            effective = _resolve_effective_device(device)
            logger.info(
                "req=%s | queue_acquired | waited_ms=%d | requested_device=%s | effective_device=%s",
                request_id,
                int(waited_s * 1000),
                _parse_device(device),
                effective,
            )

            t_ocr_start = time.perf_counter()
            result = await asyncio.to_thread(
                process_pdf_bytes_ocr_easyocr_always_images,
                pdf_bytes,
                opts,
                request_id=request_id,
                device=device,
            )
            ocr_s = time.perf_counter() - t_ocr_start

            logger.info(
                "req=%s | ocr_done | pages=%d | ocr_ms=%d | device=%s",
                request_id,
                len(result.get("pages", [])),
                int(ocr_s * 1000),
                result.get("device", "?"),
            )
            return result

    return await asyncio.wait_for(_do(), timeout=APP_TIMEOUT_SECONDS)


@app.get("/live")
async def live():
    return {"status": "ok"}


@app.get("/ready")
async def ready():
    effective = _resolve_effective_device(DEFAULT_DEVICE)

    if effective == "gpu":
        ready_state = READER_GPU is not None
    else:
        ready_state = READER_CPU is not None

    if not ready_state:
        raise HTTPException(status_code=503, detail="Not ready")

    return {
        "status": "ok",
        "gpu_available": GPU_AVAILABLE,
        "device": "cuda" if effective == "gpu" else "cpu",
    }


@app.post("/extract-text-ocr")
async def extract_text_ocr(request: Request, file: UploadFile = File(...)):
    request_id = uuid.uuid4().hex[:8]

    device = request.query_params.get("device") or request.headers.get("X-OCR-Device") or DEFAULT_DEVICE
    device = _parse_device(device)

    if file.content_type not in (
        "application/pdf",
        "application/x-pdf",
        "application/octet-stream",
    ):
        logger.warning("req=%s | bad_content_type | content_type=%s", request_id, file.content_type)
        raise HTTPException(status_code=415, detail=f"Ожидается PDF, получено: {file.content_type}")

    pdf_bytes = await file.read()
    if not pdf_bytes:
        logger.warning("req=%s | empty_file | filename=%s", request_id, file.filename)
        raise HTTPException(status_code=400, detail="Пустой файл")

    if len(pdf_bytes) > MAX_PDF_BYTES:
        logger.warning(
            "req=%s | too_large | filename=%s | size=%d | max=%d",
            request_id,
            file.filename,
            len(pdf_bytes),
            MAX_PDF_BYTES,
        )
        raise HTTPException(status_code=413, detail=f"Файл слишком большой (>{MAX_PDF_BYTES} bytes)")

    logger.info(
        "req=%s | request_start | path=%s | filename=%s | size=%d | requested_device=%s | gpu_available=%s",
        request_id,
        str(request.url.path),
        file.filename,
        len(pdf_bytes),
        device,
        GPU_AVAILABLE,
    )

    t_total = time.perf_counter()
    try:
        result = await process_pdf_with_queue_and_timeout(
            pdf_bytes,
            opts=OcrOptions(),
            request_id=request_id,
            device=device,
        )
    except asyncio.TimeoutError:
        total_ms = int((time.perf_counter() - t_total) * 1000)
        logger.error("req=%s | timeout | total_ms=%d | timeout_s=%d", request_id, total_ms, APP_TIMEOUT_SECONDS)
        raise HTTPException(status_code=504, detail="Таймаут")
    except Exception as e:
        total_ms = int((time.perf_counter() - t_total) * 1000)
        logger.exception("req=%s | error | total_ms=%d | err=%s", request_id, total_ms, str(e))
        raise HTTPException(status_code=500, detail=str(e))

    total_ms = int((time.perf_counter() - t_total) * 1000)
    logger.info("req=%s | request_done | total_ms=%d | device=%s", request_id, total_ms, result.get("device", "?"))

    return JSONResponse({"filename": file.filename, **result})


class OcrService(ocr_pb2_grpc.OcrServiceServicer):
    async def ExtractTextBidi(self, request_iterator, context: "grpc.aio.ServicerContext"):
        request_id = uuid.uuid4().hex[:8]
        filename = ""
        content_type = ""
        dpi = DPI
        device = DEFAULT_DEVICE

        buf = bytearray()

        pong_q: asyncio.Queue = asyncio.Queue()
        out_q: asyncio.Queue = asyncio.Queue()
        end_event = asyncio.Event()

        effective0 = _resolve_effective_device(device)
        await out_q.put(
            ocr_pb2.ServerMsg(
                ready=ocr_pb2.Ready(
                    request_id=request_id,
                    gpu=(effective0 == "gpu"),
                    device="cuda" if (effective0 == "gpu") else "cpu",
                )
            )
        )

        async def reader():
            nonlocal request_id, filename, content_type, dpi, device
            try:
                async for msg in request_iterator:
                    if msg.HasField("start"):
                        if msg.start.request_id:
                            request_id = msg.start.request_id
                        if msg.start.filename:
                            filename = msg.start.filename
                        if msg.start.content_type:
                            content_type = msg.start.content_type
                        if msg.start.dpi and msg.start.dpi > 0:
                            dpi = int(msg.start.dpi)
                        if msg.start.device:
                            device = msg.start.device

                    elif msg.HasField("chunk"):
                        data = msg.chunk.data
                        if data:
                            buf.extend(data)
                            if len(buf) > MAX_PDF_BYTES:
                                await context.abort(
                                    grpc.StatusCode.RESOURCE_EXHAUSTED,
                                    f"PDF too large (>{MAX_PDF_BYTES} bytes)",
                                )

                    elif msg.HasField("ping"):
                        pong_q.put_nowait(
                            ocr_pb2.ServerMsg(
                                pong=ocr_pb2.Pong(
                                    id=msg.ping.id,
                                    ts_ms=int(time.time() * 1000),
                                )
                            )
                        )

                    elif msg.HasField("end"):
                        end_event.set()
                        return

                await context.abort(grpc.StatusCode.INVALID_ARGUMENT, "Stream closed without End")
            finally:
                end_event.set()

        async def ocr_job():
            await end_event.wait()

            if not buf:
                await context.abort(grpc.StatusCode.INVALID_ARGUMENT, "Empty PDF")

            if content_type and content_type not in (
                "application/pdf",
                "application/x-pdf",
                "application/octet-stream",
            ):
                await context.abort(
                    grpc.StatusCode.INVALID_ARGUMENT,
                    f"Expected PDF content_type, got: {content_type}",
                )

            await out_q.put(ocr_pb2.ServerMsg(progress=ocr_pb2.Progress(stage="queued", pages_done=0)))

            result = await process_pdf_with_queue_and_timeout(
                bytes(buf),
                opts=OcrOptions(dpi=dpi),
                request_id=request_id,
                device=device,
            )

            await out_q.put(
                ocr_pb2.ServerMsg(
                    progress=ocr_pb2.Progress(stage="ocr_done", pages_done=len(result.get("pages", [])))
                )
            )

            await out_q.put(
                ocr_pb2.ServerMsg(
                    result=ocr_pb2.Result(
                        request_id=request_id,
                        engine=result.get("engine", "easyocr"),
                        gpu=bool(result.get("gpu", False)),
                        device=result.get("device", "cpu"),
                        dpi=int(result.get("dpi", dpi)),
                        languages=list(result.get("languages", [])),
                        text=result.get("text", ""),
                        filename=filename,
                    )
                )
            )

        reader_task = asyncio.create_task(reader())
        ocr_task = asyncio.create_task(ocr_job())

        pong_task: asyncio.Task | None = None
        out_task: asyncio.Task | None = None

        try:
            pong_task = asyncio.create_task(pong_q.get())
            out_task = asyncio.create_task(out_q.get())

            while True:
                done, _ = await asyncio.wait({pong_task, out_task}, return_when=asyncio.FIRST_COMPLETED)

                if pong_task in done:
                    msg = pong_task.result()
                    pong_task = asyncio.create_task(pong_q.get())
                else:
                    msg = out_task.result()
                    out_task = asyncio.create_task(out_q.get())

                yield msg

                if msg.HasField("result"):
                    break

            await reader_task
            await ocr_task

        except grpc.RpcError:
            raise
        except asyncio.TimeoutError:
            await context.abort(grpc.StatusCode.DEADLINE_EXCEEDED, "Timeout")
        except Exception as e:
            logger.exception("grpc req=%s | error=%s", request_id, str(e))
            await context.abort(grpc.StatusCode.INTERNAL, str(e))
        finally:
            for task in (pong_task, out_task):
                if task is not None and not task.done():
                    task.cancel()
                    with contextlib.suppress(asyncio.CancelledError):
                        await task

            for t in (reader_task, ocr_task):
                t.cancel()
                with contextlib.suppress(asyncio.CancelledError):
                    await t


async def serve_grpc():
    from grpc_reflection.v1alpha import reflection

    server = grpc.aio.server(
        options=[
            ("grpc.max_receive_message_length", MAX_PDF_BYTES + 1024 * 1024),
            ("grpc.max_send_message_length", MAX_PDF_BYTES + 1024 * 1024),
            ("grpc.keepalive_time_ms", 10_000),
            ("grpc.keepalive_timeout_ms", 5_000),
        ]
    )
    ocr_pb2_grpc.add_OcrServiceServicer_to_server(OcrService(), server)

    service_names = (
        ocr_pb2.DESCRIPTOR.services_by_name["OcrService"].full_name,
        reflection.SERVICE_NAME,
    )
    reflection.enable_server_reflection(service_names, server)

    addr = f"{GRPC_HOST}:{GRPC_PORT}"
    server.add_insecure_port(addr)
    logger.info("gRPC: starting on %s", addr)
    await server.start()
    await server.wait_for_termination()


async def _serve_http():
    import uvicorn

    config = uvicorn.Config(app, host="0.0.0.0", port=8000, log_level="info")
    server = uvicorn.Server(config)
    await server.serve()


async def main():
    await asyncio.gather(_serve_http(), serve_grpc())


if __name__ == "__main__":
    asyncio.run(main())