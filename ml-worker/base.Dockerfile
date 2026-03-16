FROM python:3.13-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    EASYOCR_MODEL_DIR=/models/easyocr

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
    -o Acquire::Retries=5 \
    -o Acquire::http::Timeout="30" \
    -o Acquire::https::Timeout="30" \
    ca-certificates \
    libglib2.0-0 \
    libgl1; \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements-base.txt /app/requirements-base.txt
RUN python -m pip install --no-cache-dir --upgrade pip && \
    python -m pip install --no-cache-dir -r /app/requirements-base.txt

RUN mkdir -p "${EASYOCR_MODEL_DIR}" && \
    python -c "import os, easyocr; d=os.environ.get('EASYOCR_MODEL_DIR','/models/easyocr'); print('Preloading EasyOCR models into:', d); easyocr.Reader(['ru','en'], gpu=False, model_storage_directory=d); print('Done')"