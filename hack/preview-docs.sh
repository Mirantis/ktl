#!/bin/sh

cd "$(dirname "$0")/.."
uvx --with mkdocs-material --with neoteroi-mkdocs mkdocs serve
