{{- $slug := replaceRE "^[0-9]{4}-[0-9]{2}-[0-9]{2}-" "" .File.ContentBaseName -}}
---
title: "{{ replace $slug "-" " " | title }}"
date: {{ .Date }}
slug: {{ $slug }}
draft: true
---
