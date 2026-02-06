# FOSDEM Website

```sh
hugo server --baseURL=http://127.0.0.1/2026 -D
```

```sh
hugo build -b https://0x51.dev/fosdem -D
pagefind --site "public"
rsync -avz --delete public/ root@0x51.dev:/var/www/0x51.dev/html/fosdem/
```

## [Pagefind](https://github.com/pagefind/pagefind)

```
pagefind --site "public"
```
