# ManageRPkgs
A simple tool to create and maintain Arch PKGBUILDS for R modules

## Usage

| flag           | use-case                                                                   | default      |
| -----------    | -----------                                                                | --------     |
| `-u`           | check for updates                                                          | false/unset  |
| `-c <PKGNAME>` | create a pkgbuild for a new package. `<PKGNAME>` should be as seen on CRAN | None         |
| `-p <file>`    | file which lists all CRAN packages which should be maintained              | `./packages` |
| `-d <folder>`   | folder where PKGBUILD folders should be stored                            | `~/src/AUR`  |


## TODO

- check for succesful clean chroot builds
- (supervised?) automatic automatic commits to AUR
- reliable License parsing

### NOTE
I'm using this to familiarize myself with golang.
Code quality should be expected to be low.
