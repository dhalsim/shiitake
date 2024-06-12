export PATH := "./node_modules/.bin:" + env_var('PATH')

run:
    postcss style.css -o bundle.css
    go build
    ./shiitake
