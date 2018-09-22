# testdata

Samples files come from [here](http://techslides.com/sample-files-for-development)

To write tags to files you can use `lltag`:

```sh
lltag sample.* \
  -a "Test Artist" \
  -t "Test Title" \
  -A "Test Album" \
  -n "3" \
  -g "Jazz" \
  -d "2000" \
  -c "Test Comment" \
  --tag ALBUMARTIST="Test AlbumArtist" \
  --tag COMPOSER="Test Composer"\
  --tag DISCNUMBER="02" \
  --tag TRACKTOTAL="06"
```
