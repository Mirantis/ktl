#!/bin/sh

cd "$(dirname "$0")/.."
uvx --with mkdocs-material mkdocs serve
