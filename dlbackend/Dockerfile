FROM pytorch/pytorch:2.4.0-cuda12.4-cudnn9-runtime

RUN apt-get update && apt-get install -y --no-install-recommends \
    nginx \
    libgl1 \
    libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY . .

# Install the DL server + load balancer. Dependencies are declared in
# pyproject.toml (there is no requirements.txt). The `gpu` extra pulls
# onnxruntime-gpu; at runtime it falls back to CPU if no GPU is present. For
# TensorRT, add the `tensorrt` extra — see dlbackend/docs/deployment.md.
RUN pip install --no-cache-dir -e ".[dl,gpu,lb]"

# Single-node stack: nginx (8899, public) → lbserver (7999) → dlserver (8001).
# nginx.conf is self-contained and proxies all traffic to 127.0.0.1:7999.
ENV LB__BACKENDS=http://127.0.0.1:8001
EXPOSE 8899

# DL_API_KEY MUST be supplied at runtime — both servers refuse to start without it:
#   docker run --gpus all -e DL_API_KEY=<secret> -p 8899:8899 dlbackend
# nginx daemonizes, lbserver runs in the background, dlserver stays in the
# foreground as the container's main process. For multi-GPU / master-slave
# scaling and auto-restart, use the Makefile targets (make start-runpod-master);
# see dlbackend/docs/deployment.md.
CMD nginx -c /app/nginx.conf \
    && (python -m lbserver --host 127.0.0.1 --port 7999 &) \
    && python -m dlserver --host 127.0.0.1 --port 8001
