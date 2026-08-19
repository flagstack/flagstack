FROM mcr.microsoft.com/devcontainers/base:ubuntu-24.04

ENV DEBIAN_FRONTEND=noninteractive
ENV LANG=en_GB.UTF-8
ENV LC_ALL=en_GB.UTF-8
ENV PNPM_HOME=/usr/local/share/pnpm
ENV PATH="${PNPM_HOME}:${PATH}"

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        bash-completion \
        build-essential \
        ca-certificates \
        curl \
        htop \
        jq \
        less \
        locales \
        make \
        postgresql-client \
        procps \
        ripgrep \
        shellcheck \
        unzip \
        vim \
        wget \
        zip \
    && locale-gen en_GB.UTF-8 \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://starship.rs/install.sh | sh -s -- -y -b /usr/local/bin

WORKDIR /workspace

CMD ["sleep", "infinity"]
