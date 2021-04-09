FROM python:3.9-slim
RUN pip install --no-cache-dir pykmip
COPY . /work
WORKDIR /work
CMD pykmip-server -f ./server.conf -l ./server.log