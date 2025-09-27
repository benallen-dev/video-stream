# video-stream

## Usage

Build frontend CSS (append `--watch` to watch for changes):

```
tailwindcss -i ./server/web/tailwind.css -o ./server/web/static/style.css
```

Run the program:
```
go run .
```

## Requirements

- Tailwindcss CLI binary (`https://github.com/tailwindlabs/tailwindcss/releases`)
    - watch functionality relies on watchman (`yay -S watchman-bin`)
