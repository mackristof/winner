FROM scratch
ADD ca-certificates.crt /etc/ssl/certs/
ADD winner /app/
EXPOSE 8000
CMD ["/app/winner"]
