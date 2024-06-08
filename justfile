run:
    BROWSERSLIST_IGNORE_OLD_DATA=true tailwindcss -i style.css -o bundle.css
    go build
    ./shiitake
