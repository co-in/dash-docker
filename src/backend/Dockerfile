FROM golang:bullseye as builder

RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN go get golang.org/x/net
RUN go build -o main .

FROM debian:bullseye

ARG USER_ID
ARG GROUP_ID
ARG BINARY

ENV HOME /dashd
ENV USER_ID ${USER_ID:-1000}
ENV GROUP_ID ${GROUP_ID:-1000}
ENV BINARY ${BINARY:-"dashpay-instant"}

RUN groupadd -g ${GROUP_ID} dashd
RUN useradd -u ${USER_ID} -g dashd -s /bin/bash -m -d /dashd dashd
RUN mkdir /dashd/.dashcore
RUN chown dashd:dashd -R /dashd

RUN apt-get update
RUN apt-get install -y locales curl ca-certificates gnupg lsb-release wget nano htop mc \
    && rm -rf /var/lib/apt/lists/* \
    && localedef -i en_US -c -f UTF-8 -A /usr/share/locale/locale.alias en_US.UTF-8

RUN DASH_VERSION=$(curl -s https://github.com/dashpay/dash/releases/latest | grep -Po '(?<=/tag/v)[^"]+') \
   && URL="https://github.com/dashpay/dash/releases/download/v${DASH_VERSION}/dashcore-${DASH_VERSION}-x86_64-linux-gnu.tar.gz" \
   && FILE=${URL##*/} \
   && DIR=${FILE%.tar.gz} \
   && wget -O /tmp/dashcore.tar.gz -q ${URL}

RUN tar -xvf /tmp/dashcore.tar.gz -C /tmp/
RUN cp /tmp/dashcore*/bin/*  /usr/local/bin
RUN rm -rf /tmp/dashcore*

COPY --from=builder /build/main /usr/local/bin/${BINARY}
RUN chmod a+x /usr/local/bin/*

EXPOSE 9998 9999 19998 19999
WORKDIR /dashd

USER "${USER_ID}:${GROUP_ID}"
