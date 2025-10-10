# Full documentation as a static site

Using
- hugo with theme [hugo-theme-relearn](https://mcshelby.github.io/hugo-theme-relearn/)
- GitLab Pages deployment with linked domain https://toop.sickit.eu

## Edit and review locally

```shell
hugo serve --source ./docs
# or
docker run --rm -it -v $PWD/docs:/src -p 1313:1313 hugomods/hugo:exts-0.150.0 serve --source /src
```
