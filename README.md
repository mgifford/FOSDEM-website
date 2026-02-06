# FOSDEM Website

```sh
hugo server --baseURL=127.0.0.1/2026 -D
```

```sh
hugo build -b https://0x51.dev/fosdem -D
rsync -avz --delete public/ root@0x51.dev:/var/www/0x51.dev/html/fosdem/
```
