FROM scratch
ADD ca-certificates.crt /etc/ssl/certs/
ADD winner /
EXPOSE 8000
CMD ["/winner"]
