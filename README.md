Span
====

Install with

    $ go get github.com/miku/span/cmd/...

or via deb or rpm [packages](https://github.com/miku/span/releases).

Formats
-------

* [CrossRef API](http://api.crossref.org/), works and members
* JATS [Journal Archiving and Interchange Tag Set](http://jats.nlm.nih.gov/archiving/versions.html), with various flavours for JSTOR and others
* [DOAJ](http://doaj.org/) exports
* FINC [Intermediate Format](https://github.com/ubleipzig/intermediateschema)
* Various FINC [SOLR Schema](https://github.com/finc/index/blob/master/schema.xml)
* GENIOS Profile XML
* Elsevier Transport
* Thieme TM Style
* [Formeta](https://github.com/culturegraph)
* IEEE IDAMS Exchange V2.0.0

Also:

* [KBART](http://www.uksg.org/KBART)

TODO
----

* Decouple format from source. Things like SourceID and MegaCollection are per source, not format.

Jsoniter testdrive
------------------

* encoding/json

```
$ time taskcat GeniosIntermediateSchema | span-tag -c $(taskoutput AMSLFilterConfig) > /dev/null
...
real    11m48.803s
user    40m15.980s
sys      0m32.880s
```

* jsoniter/go

```
$ time taskcat GeniosIntermediateSchema | span-tag -c $(taskoutput AMSLFilterConfig) > /dev/null
...

real     9m25.871s
user    31m29.240s
sys      0m32.572s
```

Licence
-------

* GPLv3
* This project uses the Compact Language Detector 2 - [CLD2](https://github.com/CLD2Owners/cld2), Apache License Version 2.0

Next steps
----------

The intermediate format consists of two kinds of fields:

* finc-independent (title, author, issn, ...)
* finc-dependent (finc id, internal source id, specific format from mappings, ...)

A catalog of input formats could be made reusable, by separating the above two concerns:

```
$ span-import -i <FORMAT> <FILE>
```

This could just a normalizer.

```
$ span-import -i <FORMAT> -finc <FILE>
```

Could add more finc-dependent fields.

```
type SomeFormat struct {}

func (f SomeFormat) Normalize() (IntermediateSchema, error)

func (f SomeFormat) FincFields() (FincFields, error)

...

// Merge the above two for output.
```
