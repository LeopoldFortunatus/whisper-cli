whisper cli
convert mp3 to m4a:
```
ffmpeg -i 2025_10_18_13_autum_plots.mp3 2025_10_18_13_autum_plots.m4a
```
mass convert in directory:
```
for f in `ls *.MP3`; do ffmpeg -i  $f ${f%.*}.m4a; done
```