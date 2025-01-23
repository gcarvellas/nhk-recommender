# NHK Recommender

This program queries [NHK News](https://www.nhk.or.jp/) and outputs each article with it's difficulty based off your available anki cards.

# Requirements
- [AnkiConnect](https://ankiweb.net/shared/info/2055492159AnkiConnect)
- go

# Usage
```
go build .
./nhk-recommender
0.732955,菅野智之「チャンピオン狙えるチームでプレーしたかった」,/news/html/20241220/k10014673431000.html
0.714953,松山英樹が優勝 男子ゴルフアメリカツアーの今季開幕戦,/news/html/20250106/k10014685651000.html
0.701754,男子ゴルフ 米ツアー第2戦 松山英樹は16位 2週連続優勝ならず,/news/html/20250113/k10014691841000.html
...
```
The output is `Difficulty,Title,URL`

You can also view program arguments with `./nhk-recommender --help`
