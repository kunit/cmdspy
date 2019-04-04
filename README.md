# cmdspy [![Build Status](https://travis-ci.org/kunit/cmdspy.svg?branch=master)](https://travis-ci.org/kunit/cmdspy) [![codecov](https://codecov.io/gh/kunit/cmdspy/branch/master/graph/badge.svg)](https://codecov.io/gh/kunit/cmdspy)

引数で指定したコマンドが実行されているかを一定間隔ごとに Slack に通知します。

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

## Slack 通知サンプル

`sh /Users/kunit/Work/test.sh` を10秒単位に通知させた結果です。

```
$ ./cmdspy -i 10 -c config.toml "sh /Users/kunit/Work/test.sh"
```
<img width="659" alt="スクリーンショット" src="https://user-images.githubusercontent.com/405750/55526937-faf49f80-56d1-11e9-9fac-c39a751733df.png">
