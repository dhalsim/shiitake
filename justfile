export PATH := "./node_modules/.bin:" + env_var('PATH')
export GTK_A11Y := "none"

run:
    npm i
    postcss style.css -o bundle.css
    go build
    ./shiitake
