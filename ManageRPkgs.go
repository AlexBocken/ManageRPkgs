package main

// TODO: Apache License -> Example : showtext

import (
	"log"
	"strings"
	"text/template"
	"time"
	"os"
	"github.com/PuerkitoBio/goquery"
	"github.com/cavaliergopher/grab/v3"
	"fmt"
	"crypto/sha256"
	"io"
	"bufio"
	"flag"
	"os/exec"
)


func cleanup(list_str string)(string){
	var acc []string
	var tmp string
	list_str = strings.ToLower(list_str)
	split_arr := strings.Split(list_str, ",")
	for _, item := range split_arr{
		if strings.ContainsAny(item, "(|≥|)"){
			tmp = strings.ReplaceAll(item, "≥", ">=")
			tmp = strings.ReplaceAll(tmp, " ", "")
			tmp = strings.ReplaceAll(tmp, "(", "")
			tmp = strings.ReplaceAll(tmp, ")", "")
			if strings.Split(tmp, ">")[0] != "r"{
				tmp = "r-" + tmp
			}
			tmp = "'" + tmp + "'"
		} else {
			if item != "r"{
				tmp = "r-" + item
			}
			tmp = strings.ReplaceAll(tmp, " ", "")
		}
		acc = append(acc, tmp)
	}
	return strings.Join(acc," ")
}


func get_sha256sum(Filename string)(string){
  	f, err := os.Open(Filename)
  	if err != nil {
  	  log.Fatal(err)
  	}
  	defer f.Close()

  	h := sha256.New()
  	if _, err := io.Copy(h, f); err != nil {
  	  log.Fatal(err)
  	}

	return fmt.Sprintf("%x", h.Sum(nil))
}


func gen_pkgbuild(cranname string, basedir string){
	type Pkgbuild struct{
		Cranname string
		Description string
		Version string
		Archive_date string
		Depends string
		Optdepends string
		Arch string
		License string
		Checksum string
	}

	var pkgbuild Pkgbuild
	pkgbuild.Cranname = cranname
	url := "https://cran.r-project.org/web/packages/" + pkgbuild.Cranname
	doc, err := goquery.NewDocument(url)

    	if err != nil {
    	    log.Fatal(err)
	    panic(err)
    	}

	var  version, depends,  publish_date string

	pkgbuild.Description = strings.SplitN(doc.Find("h2").Text(), ": ", 2)[1]
	trs := doc.Find("tr")
	for i := range trs.Nodes {
		children := trs.Eq(i).Children()
		category_text := children.Eq(0).Text()
		switch category_text {
		case "Version:":
			version = children.Eq(1).Text()
		case "Depends:", "Imports:":
			// add commas between Depends and Imports for clean string splitting
			if depends != ""{
				depends += ","
			}
			depends += children.Eq(1).Text()
		case "Suggests:":
			pkgbuild.Optdepends = cleanup(children.Eq(1).Text())
		case "Published:":
			publish_date = children.Eq(1).Text()
		case "NeedsCompilation:":
			if children.Eq(1).Text() == "yes"{
				pkgbuild.Arch="x86_64"
			} else if children.Eq(1).Text() == "no"{
				pkgbuild.Arch="any"
			}
		case "License:":
			tmp := strings.ReplaceAll(children.Eq(1).Text(), "-", "")
			tmp = strings.ReplaceAll(tmp, " | ", " ")
			tmp = strings.ReplaceAll(tmp, "+ file LICENSE", "custom")
			tmp = strings.ReplaceAll(tmp, "Apache License (≥ 2.0)", "Apache")
			pkgbuild.License = tmp
    		default:
		}

	}


	pkgbuild.Version = version
	pkgbuild.Depends = cleanup(depends)
	layout := "2006-01-02"
	t, _ := time.Parse(layout, publish_date)
	t = t.AddDate(0, 0, 1)
	archive_date := t.Format(layout)
	pkgbuild.Archive_date = archive_date

	url = "https://cran.microsoft.com/snapshot/" + pkgbuild.Archive_date + "/src/contrib/" + pkgbuild.Cranname + "_" + pkgbuild.Version + ".tar.gz"
	response, error := grab.Get(".", url)
	if error != nil {
    		log.Fatal(err)
	}

	pkgbuild.Checksum = get_sha256sum(response.Filename)

	const final_pkgbuild= `# Maintainer: Alexander Bocken <alexander@bocken.org>

_cranname={{.Cranname}}
_cranver={{.Version}}
_archivedate={{.Archive_date}}
pkgname=r-${_cranname,,}
pkgver=${_cranver//[:-]/.}
pkgrel=1
pkgdesc="{{.Description}}"
arch=({{.Arch}})
url="https://cran.r-project.org/package=${_cranname}"
license=({{.License}})
depends=({{.Depends}})
optdepends=({{.Optdepends}})
source=("https://cran.microsoft.com/snapshot/${_archivedate}/src/contrib/${_cranname}_${_cranver}.tar.gz")
sha256sums=({{.Checksum}})

build() {
  R CMD INSTALL ${_cranname}_${_cranver}.tar.gz -l "${srcdir}"
}

package() {
  install -dm0755 "${pkgdir}/usr/lib/R/library"

  cp -a --no-preserve=ownership "${_cranname}" "${pkgdir}/usr/lib/R/library"
}
`
	tmpl := template.Must(template.New("test").Parse(final_pkgbuild))
	folder_name := basedir + "r-" + strings.ToLower(cranname)
	f, err := os.OpenFile(folder_name + "/PKGBUILD", os.O_WRONLY|os.O_CREATE, 0644)
	err = tmpl.Execute(f, pkgbuild)
	if err != nil {
		log.Println("executing template:", err)
	}
	update_SRCINFO(folder_name)
}


func ReadLine(r io.Reader, lineNum int) (line string, lastLine int, err error) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		lastLine++
	    	if lastLine == lineNum {
	    		// you can return sc.Bytes() if you need output in []bytes
	    	    	return sc.Text(), lastLine, sc.Err()
	    	}
	}
	return line, lastLine, io.EOF
}

func GetRemoteVersion(package_name string)(string){
	doc, err := goquery.NewDocument("https://cran.r-project.org/web/packages/" + package_name)
	if err != nil{
		log.Panic("could not load url, err:", err)
	}

	trs := doc.Find("tr")
	for i := range trs.Nodes {
		children := trs.Eq(i).Children()
		category_text := children.Eq(0).Text()
		switch category_text {
		case "Version:":
			return children.Eq(1).Text()
		default:
		}
	}
	return ""
}

func check_for_updates(packages []string, base_dir string)([]string){
	var need_updates []string
	for _, package_name := range packages{
		fmt.Println("Checking updates for", package_name)
		folder_name := base_dir  + "r-" + strings.ToLower(package_name)
		r, _ := os.Open(folder_name + "/PKGBUILD")
		line, _, _ := ReadLine(r, 3)
		cranname := strings.Split(line, "=")[1]
		r, _ = os.Open(folder_name + "/PKGBUILD")
		line, _, _ = ReadLine(r, 4)
		cranver_local := strings.Split(line, "=")[1]

		cranver_remote := GetRemoteVersion(cranname)
		fmt.Println("Local version:", cranver_local, "Remote version:", cranver_remote)
		if  cranver_remote != cranver_local{
			need_updates = append(need_updates, package_name)
		}
	}
	return need_updates
}


func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
    	if err != nil {
    		return nil, err
    	}
    	defer file.Close()

    	var lines []string
    	scanner := bufio.NewScanner(file)
    	for scanner.Scan() {
    		lines = append(lines, scanner.Text())
    	}
    return lines, scanner.Err()
}

func update_routine(file string, basedir string){
	names, _ := readLines(file)
	packages := check_for_updates(names, basedir)
	for _, package_name := range packages{
		fmt.Println("Updating PKGBUILD for", package_name)
		gen_pkgbuild(package_name, basedir)
	}
	if len(packages) == 0 {
		fmt.Println("All packages are up to date")
	} else{
		fmt.Println("Done")
	}
}

func append_to_file(filename string, newline string){
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
	    panic(err)
	}

	defer f.Close()

	if _, err = f.WriteString(newline + "\n"); err != nil {
	    panic(err)
	}
}

func update_SRCINFO(dirname string){
	fmt.Println("Updating... .SRCINFO")
	start, _ := os.Getwd()
	os.Chdir(dirname)
	fmt.Println(os.Getwd())
	cmd := exec.Command("makepkg", "--printsrcinfo", ">", ".SRCINFO")
    	_ , err := cmd.Output()
	if err != nil{
		log.Fatal("Error at creating SRCINFO in ", dirname, " Error:", err)
		panic(err)
	}
	os.Chdir(start)
	fmt.Println(os.Getwd())
}

func main() {
	home_dir, _ := os.UserHomeDir()

	doUpdatePtr := flag.Bool("u", false, "Check for updates")
	packagesPtr := flag.String("p", "packages", "file listing all crannames")
	newPkgNamePtr := flag.String("c", "", "Create a new Pkgbuild for package")
	baseDirPtr := flag.String("d",  home_dir + "/src/AUR/", "Directory with all Package folders")
	flag.Parse()
	if *newPkgNamePtr != ""{
		fmt.Println("Creating folder", *newPkgNamePtr)
		os.Mkdir(*baseDirPtr + "/r-"+strings.ToLower(*newPkgNamePtr), os.FileMode(int(0755)))
		append_to_file(*packagesPtr, *newPkgNamePtr)
		gen_pkgbuild(*newPkgNamePtr, *baseDirPtr)
	}
	if *doUpdatePtr{
		update_routine(*packagesPtr, *baseDirPtr)
	}

}
