FROM scratch
ADD winner /
EXPOSE 8000
CMD ["/winner"]
