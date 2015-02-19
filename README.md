Span
====

Span formats.

Docs: http://godoc.org/github.com/miku/span

[![Build Status](https://travis-ci.org/miku/span.svg?branch=master)](https://travis-ci.org/miku/span)

Formats
-------

* [CrossRef API](http://api.crossref.org/)
* [OVID](http://rzblx4.uni-regensburg.de/ezeitdata/admin/ezb_export_ovid_v01.xsd)
* Finc

Usage
-----

    $ span
    Usage: span [OPTIONS] CROSSREF.LDJ
      -allow-empty-institutions=false: keep records, even if no institutions is using it
      -b=25000: batch size
      -cpuprofile="": write cpu profile to file
      -hspec="": ISIL PATH pairs
      -hspec-export=false: export a single combined holdings map as JSON
      -ignore=false: skip broken input record
      -members="": path to LDJ file, one member per line
      -v=false: prints current program version
      -verbose=false: print debug messages
      -w=4: workers

----

**Inputs**

* An input LDJ containing all crossref works metadata, one [crossref.Document](https://github.com/miku/span/blob/5585dc500d82fcab9c783937d7d567fdffb71fde/crossref/document.go#L46) per line. [Example API response](http://api.crossref.org/works/56).

Optionally:

* A number of XML files, containing holdings information for various institutions in [OVID](http://rzblx4.uni-regensburg.de/ezeitdata/admin/ezb_export_ovid_v01.xsd) format.
* A file containing information about [members](https://github.com/miku/span/blob/aa59d6468bad530fbf680c529e341b76e033386c/crossref/api.go#L23), in LDJ format. [Example API response](http://api.crossref.org/members/56).

One can transform the documents with the `span` tool:

    span -hspec DE-15:file.xml,DE-20:other.xml crossref.ldj

Additionally, if one has a cached file of members API responses, one can
use it as input. This way the API does not need to be called at all:

    span -hspec DE-15:file.xml,DE-10:other.xml -members members.ldj crossref.ldj

The output is an LDJ in [finc.SolrSchema](https://github.com/miku/span/blob/aa59d6468bad530fbf680c529e341b76e033386c/finc/schema.go#L5).
