FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    afl++ clang python3 ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /opt/zero-day
COPY demo/target.c ./target.c
RUN AFL_LLVM_LAF_ALL=1 afl-clang-fast -O1 -g -o /opt/zero-day/target target.c \
    && mkdir /work && chown 1000:1000 /work
COPY campaign.py guidance.example.json ./
USER 1000:1000
ENV HOME=/tmp PYTHONDONTWRITEBYTECODE=1
ENTRYPOINT ["python3", "/opt/zero-day/campaign.py"]
CMD ["run", "--seconds", "30", "--guidance", "/opt/zero-day/guidance.example.json"]
