FROM --platform=linux/x86-64 alpine:3.12

MAINTAINER Prasad Ghangal<prasad.ghangal@gmail.com>

ADD covaccine-notifier /covaccine-notifier
ENTRYPOINT ["/covaccine-notifier"]
