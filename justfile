run:
    tailwindcss -i style.css -o bundle.css
    go build
    ./shiitake
