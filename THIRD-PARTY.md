# Third-party components

The FOSDEM website bundles and generates code written by other people. This
file records what is included, under which licence, and where its licence text
lives. It supplements `LICENSE`, which covers FOSDEM's own software and content
only.

Every licence below is the verbatim upstream file, kept next to the component
it applies to.

## Bundled in this repository

Committed under `static/` and served to every visitor.

| Component | Version | Licence | Licence text |
|-----------|---------|---------|--------------|
| Leaflet | 1.9.4 | BSD-2-Clause | `static/css/leaflet/LICENSE` |
| Signika | 2018 | SIL OFL 1.1 | `static/css/fonts/Signika/OFL.txt` |
| DejaVu Sans | 2.37 | Bitstream Vera and Arev licences, plus public domain changes | `static/css/fonts/dejavu-sans-ttf-2.37/LICENSE` |

## Generated into the published site

`pagefind --site public` writes its search runtime into the build output, so
Pagefind's code is redistributed as part of the deployed site even though it is
not committed here. Its licence is kept under `licenses/` for that reason.

| Component | Version | Licence | Licence text |
|-----------|---------|---------|--------------|
| Pagefind | 1.4.0 | MIT | `licenses/pagefind-LICENSE` |

## Map data

The homepage map loads tiles from OpenStreetMap. Map data is
© OpenStreetMap contributors, available under the Open Database License. The
attribution is rendered on the map itself in `layouts/home.html`.

## Build tools

These produce the site but are never redistributed with it, so they impose no
attribution requirement. Listed for information.

| Tool | Licence |
|------|---------|
| Hugo | Apache-2.0 |
| Go | BSD-3-Clause |
| Nix | LGPL-2.1-or-later |
