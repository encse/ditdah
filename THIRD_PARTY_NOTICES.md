# Third-party notices

## Hamlib

DitDah includes Hamlib 4.7.2, statically linked into the executable.

The Hamlib frontend and backend libraries are licensed under the GNU Lesser
General Public License, version 2.1 or (at your option) any later version.
Hamlib's original `LICENSE`, `COPYING`, `COPYING.LIB`, `AUTHORS`, and
`README.md` files are preserved under `licenses/hamlib/` in every DitDah
release archive.

The complete, unmodified Hamlib 4.7.2 source archive is published alongside
the DitDah binaries in the corresponding GitHub release and is also available
from:

https://github.com/Hamlib/Hamlib/releases/download/4.7.2/hamlib-4.7.2.tar.gz

Its SHA-256 checksum is:

`ae1fcf2dbc80ea0786ea8f047b09399c3f7737d1930442f61a031708ed33e88f`

The complete DitDah source for each binary is available from the matching Git
tag at:

https://github.com/encse/ditdah

The source includes `scripts/build-hamlib.sh`, which accepts
`DITDAH_HAMLIB_SOURCE_DIR` to build and relink DitDah with a modified Hamlib
source tree.
