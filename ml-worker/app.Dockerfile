# syntax=docker/dockerfile:1.7

FROM cr.yandex/crpgiggf30987ecj5lp4/python-easyocr-gpu-base:cu126-v1

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    EASYOCR_MODEL_DIR=/models/easyocr

WORKDIR /app

COPY requirements-app.txt /app/requirements-app.txt

RUN --mount=type=cache,target=/root/.cache/pip \
    python -m pip install --no-cache-dir -r /app/requirements-app.txt

COPY main.py /app/main.py
COPY ocr_pb2.py /app/ocr_pb2.py
COPY ocr_pb2_grpc.py /app/ocr_pb2_grpc.py

EXPOSE 8000 50051

CMD ["python", "-u", "/app/main.py"]