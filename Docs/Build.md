
Запуск скрипта генерации иконки

docker run --rm \
  -v "$PWD":/app \
  -w /app \
  --user "$(id -u):$(id -g)" \
  -e HOME=/tmp \
  -e PIP_CACHE_DIR=/tmp/pip-cache \
  -e PIP_NO_CACHE_DIR=1 \
  python:3.12-slim \
  sh -c "pip install --quiet --target /tmp/pylibs Pillow && PYTHONPATH=/tmp/pylibs python tools/genicon.py"