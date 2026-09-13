FROM debian:bookworm

RUN apt-get update && apt-get install -y \
    git \
    build-essential \
    autoconf \
    automake \
    libtool \
    pkg-config \
    ragel \
    libev-dev \
    liburcu-dev \
    libsodium-dev \
    libunwind-dev \
    libmaxminddb-dev \
    ca-certificates \
    dnsutils \
    netcat-openbsd \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src

RUN git clone --depth 1 --branch 3.9.x https://github.com/gdnsd/gdnsd.git .

RUN autoreconf -fi

RUN ./configure \
    --prefix=/usr \
    --sysconfdir=/etc \
    --localstatedir=/var

RUN make -j"$(nproc)" \
    && make install

RUN useradd \
    --system \
    --home /var/run/gdnsd \
    --shell /usr/sbin/nologin \
    gdnsd

RUN mkdir -p /etc/gdnsd /var/run/gdnsd /var/log/gdnsd \
    && chown -R gdnsd:gdnsd /var/run/gdnsd /var/log/gdnsd

EXPOSE 53/tcp
EXPOSE 53/udp
