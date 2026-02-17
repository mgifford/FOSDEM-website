# FOSDEM Website

You will need to fetch `schedule.json` from Pretalx here:
https://pretalx.fosdem.org/fosdem-2026/schedule/export/schedule.json

Note that this link only works if you are logged in to Pretalx, else
you will get a 404.

```sh
hugo server --baseURL=http://127.0.0.1/2026 -D
```

```sh
go run main.go
hugo build -b https://0x51.dev/fosdem -D
pagefind --site "public"
rsync -avz --delete public/ root@0x51.dev:/var/www/0x51.dev/html/fosdem/
```

## [Pagefind](https://github.com/pagefind/pagefind)

```
pagefind --site "public"
```
