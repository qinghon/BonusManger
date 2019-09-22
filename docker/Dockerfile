FROM alpine:latest
RUN mkdir -p /opt/bcloud/ &&touch /opt/bcloud/node.db &&apk add lvm2 curl wget
COPY build/bonus_manger_x86_64 /bonus_manger
COPY node_x86_64 /node
COPY start.sh /

CMD ["sh","/start.sh"]