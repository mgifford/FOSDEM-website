---
title: "Example post with an image"
date: 2026-08-31
slug: example-post-with-image
image: solbosch-campus.png
draft: true
---

This post is a draft, so it is excluded from `hugo build` and from the RSS feed.
Preview it with `hugo server --config hugo.landing.yaml -D`. It exists to
exercise every element a news post can contain.

## Images

Images live next to `index.md` in the post directory. Reference them by
filename. The render hook resolves them as page resources, generates a `srcset`
capped to the article width, and emits `width`/`height` to stop the layout
shifting.

![The ULB Solbosch campus](solbosch-campus.png "Adding a quoted title turns the image into a figure with this caption.")

Without the quoted title it renders as a bare image:

![The ULB Solbosch campus](solbosch-campus.png)

The feed picks a featured image for `enclosure`, `media:content` and
`media:thumbnail`, and the page reuses it for `og:image`. It defaults to the
first image in the directory. Set `image:` in front matter to choose another.

## Text

Body text renders in DejaVu Sans. **Bold** and *italic* both work, as does
`inline code`, a [link](https://fosdem.org/), and ~~strikethrough~~.

### Third level heading

#### Fourth level heading

Lists keep the body font:

- First item
- Second item, long enough to wrap onto a second line so the hanging indent is
  visible against the paragraph above it
- Third item
  - Nested item

1. Numbered item
2. Another numbered item

> A blockquote sits flush with the text column rather than taking the browser
> default side margins.

Code blocks scroll horizontally instead of stretching the article:

```yaml
outputs:
  home:
    - HTML
    - RSS
  section:
    - HTML
```

| Room       | Capacity | Building |
| ---------- | -------- | -------- |
| Janson     | 1400     | K        |
| K.1.105    | 400      | K        |
| UB2.252A   | 100      | U        |

---

That horizontal rule and this closing paragraph are the last of it.
