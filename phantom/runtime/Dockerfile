FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends tor wireguard-tools iproute2 iptables python3 curl ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY artifacts/phantom-proxy /usr/local/bin/phantom-proxy
COPY torrc /etc/tor/torrc
COPY wireguard.py /opt/phantom/wireguard.py
RUN mkdir -p /var/lib/tor && chown debian-tor:debian-tor /var/lib/tor && chmod 700 /var/lib/tor
USER debian-tor
CMD ["tor", "-f", "/etc/tor/torrc"]
