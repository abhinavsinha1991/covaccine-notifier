FROM --platform=linux/x86-64 alpine:3.12

MAINTAINER Abhinav S<sinha.abhinav1991@gmail.com>

ADD covaccine-notifier /covaccine-notifier
ENTRYPOINT ["/covaccine-notifier"]
