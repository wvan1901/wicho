# devlog
A slog handler that provides colorful logging for local development ONLY.

## Objective
* Create a slog handler that prints logs with useful colors

## Requirements
* No external library's
* All logs must be visuallly pleasing
* Uses Ansi commands

## Non requirements
* Doesn't have to be fast
* Doesn't have to be memory efficient

# Options
To customize your logger experince you can modify these options in the Options struct:
* Level
* AddSource

`NOTE:`When initilizing the logger you can also pass a custom theme to customize your color theme.\
If nil then it will use the default theme

## Resources used
* [Slog color handler guide](https://dusted.codes/creating-a-pretty-console-logger-using-gos-slog-package)
* [Golang slog guide](https://github.com/golang/example/tree/master/slog-handler-guide)
* [Ansi helper](https://stackoverflow.com/questions/4842424/list-of-ansi-color-escape-sequences)
