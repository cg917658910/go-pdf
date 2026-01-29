build:
	go build ./cmd/pdf-guard
create:
	./pdf-guard --in a.pdf --out c.pdf --start 2026-01-29T11:41:00Z --end 2026-01-29T12:05:00Z
cp:
	cp c.pdf /mnt/e/code
.PHONY: build create cp run
run: build create cp
	@echo "run: build, create, cp completed"
