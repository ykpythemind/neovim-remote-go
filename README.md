# neovim-remote-go

alternative to https://github.com/mhinz/neovim-remote

## Install


```
$ go install github.com/ykpythemind/neovim-remote-go/cmd/neovim-remote
```


## Usage

```
neovim-remote --remote-wait go.mod
```

### Use neovim-remote as git editor.

see https://github.com/mhinz/neovim-remote/blob/master/README.md

```.vimrc
if has('nvim')
  let $GIT_EDITOR = 'nvr -cc split --remote-wait'
endif

autocmd FileType gitcommit,gitrebase,gitconfig set bufhidden=delete
```

## TODO

- some opts https://github.com/mhinz/neovim-remote/blob/1ec7c6c76a66d8d381f54570d6fdd3079c190ba5/nvr/nvr.py#L218
- test
