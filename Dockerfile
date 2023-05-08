FROM alpine:latest

ARG PB_VERSION=0.15.2

RUN apk add --no-cache \
    unzip \
    # this is needed only if you want to use scp to copy later your pb_data locally
    openssh

# download and unzip PocketBase
ADD https://github.com/TheRedSpy15/dietly-pb/releases/download/${PB_VERSION}/gin.zip /tmp/pb.zip
RUN unzip /tmp/pb.zip -d /pb/

EXPOSE 8080

CMD ["./dietly-pb", "serve", "--http=0.0.0.0:8080", "--encryptionEnv=PB_ENCRYPTION_KEY"]
