FROM scratch

ADD --chmod=755 https://unpkg.com/@esbuild/linux-x64@0.19.2/bin/esbuild /

WORKDIR /assets

ENTRYPOINT [ "/esbuild",                          \
             "--asset-names=[dir]/[name]-[hash]", \
             "--bundle",                          \
             "--entry-names=[dir]/[name]-[hash]", \
             "--loader:.jpg=copy",                \
             "--loader:.png=copy",                \
             "--loader:.svg=copy",                \
             "--metafile=assets.json",            \
             "--minify",                          \
             "--outbase=.",                       \
             "--outdir=dist",                     \
             "--public-path=/dist",               \
             "--target=chrome88",                 \
             "css/**", "img/**", "js/**" ]
