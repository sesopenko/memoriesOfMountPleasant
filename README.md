# Memories of Mount Pleasant


### Docker Run Example

This example will mount the images at `/mnt/dest` path.

```bash
docker build -t mountpleasant .
```


```bash
docker run -it \
  --mount type=bind,source="/mnt/sean-documents/art concept ai/memories_of_mount_pleasant",target="/mnt/dest",readonly \
  -e IMAGE_PATH=/mnt/dest \
  -p 8080:8080 \
  mountpleasant
```