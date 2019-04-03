# cmdspy [![Build Status](https://travis-ci.org/kunit/cmdspy.svg?branch=master)](https://travis-ci.org/kunit/cmdspy) [![codecov](https://codecov.io/gh/kunit/cmdspy/branch/master/graph/badge.svg)](https://codecov.io/gh/kunit/cmdspy)

引数で指定したコマンドが実行されているかを一定間隔ごとに Slack に通知します

## 使い方

```
$ cmdspy --help
Usage: cmdspy [--version] [--help] <options> "command <arg1> <arg2>..."

Options:
  -c, --config                             path to configuration file
  -i, --interval=seconds                   report interval (default: 600)

% echo config.toml
url = "https://hooks.slack.com/services/XXXXXX/YYYYYYY/ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ"
channel = "#chanel_name"
emoji = ":emoji_name:"
interval = 1800

% cmdspy -c config.toml "/path/to/long_time_cmd -a -b -c"
```