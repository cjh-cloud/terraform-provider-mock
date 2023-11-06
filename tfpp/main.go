package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func main() {
	log.Println("📦 Packaging Terraform Provider for private registry...")

	namespace := flag.String("ns", "", "Namespace for the Terraform registry.") // No default
	domain := flag.String("d", "", "Private Terraform registry domain.") // No default
	providerName := flag.String("p", "mock", "Name of the Terraform provider.")
	distPath := flag.String("dp", "../dist", "Path to Go Releaser build files.")
	repoName := flag.String("r", "terraform-provider-mock", "Name of the provider repository used in Go Releaser build name.")
	version := flag.String("v", "0.0.1", "Semantic version of build.")
	gpgFingerprint := flag.String("gf", "", "GPG Fingerprint of key used by Go Releaser") // No default
	gpgPubKeyFile := flag.String("gk", "../key.txt", "Path to GPG Public Key in ASCII Armor format.")
	flag.Parse()

	*distPath = *distPath + "/" // If the path already ends in "/", it shouldn't matter

	// Create release dir - only the contents of this need to be uploaded to S3
	err := createDir("release")
	if err != nil {
		log.Printf("Error creating 'release' dir: %s", err)
	}

	// Create .wellKnown directory and terraform.json file
	err = wellKnown()
	if err != nil {
		log.Printf("Error creating '.wellKnown' dir: %s", err)
	}

	// Create v1 directory
	err = provider(*namespace, *providerName, *distPath, *repoName, *version, *gpgFingerprint, *gpgPubKeyFile, *domain)
	if err != nil {
		log.Printf("Error creating 'v1' dir: %s", err)
	}

	log.Println("📦 Packaged Terraform Provider for private registry.")
}

// This establishes the "API" as a TF provider by responding with the correct JSON payload, by using static files
func wellKnown() (error) {
	log.Println("* Creating .well-known directory")

	err := createDir("release/.well-known")
	if err != nil {
		return err
	}

	terraformJson := []byte(`{"providers.v1": "/v1/providers/"}`)

	log.Println("  - Writing to .well-known/terraform.json file")
	err = writeFile("release/.well-known/terraform.json", terraformJson)
	if err != nil {
		return err
	}

	return nil
}

// provider is the Terraform name
// repoName is the Repository name
func provider(namespace, provider, distPath, repoName, version, gpgFingerprint, gpgPubKeyFile, domain string) (error) {
	// Path to semantic version dir
	versionPath := providerDirs(namespace, provider, version)

	// Files to create under v1/providers/[namespace]/[provider_name]
	createVersionsFile(namespace, provider, distPath, repoName, version) // Creates version file one above download, which is why downloadPath isn't used

	// Files/Directories to create under v1/providers/[namespace]/[provider_name]/[version]
	copyShaFiles(versionPath, distPath, repoName, version)
	downloadPath, err :=  createDownloadsDir(versionPath)
	if err != nil {
		return err
	}

	// Create darwin, freebsd, linux, windows dirs
	createTargetDirs(*downloadPath)

	// Copy all zips
	copyBuildZips(*downloadPath, distPath, repoName, version)

	// Create all individual files for build targets and each architecture for the build targets
	createArchitectureFiles(namespace, provider, distPath, repoName, version, gpgFingerprint, gpgPubKeyFile, domain)

	return nil
}

// Create the directories with a path format v1/providers/[namespace]/[provider_name]/[version]
func providerDirs(namespace, repoName, version string) string {
	log.Println("* Creating release/v1/providers/[namespace]/[repo]/[version] directories")

	providerPathArr := [6]string{"release", "v1", "providers", namespace, repoName, version}

	var currentPath string
	for _, v := range providerPathArr {
		currentPath = currentPath + v + "/"
		createDir(currentPath)
	}

	return currentPath
}

// Create the versions file under v1/providers/[namespace]/[provider_name]
func createVersionsFile(namespace, provider, distPath, repoName, version string) (error) {
	log.Println("* Writing to release/v1/providers/[namespace]/[repo]/versions file")

	versionPath := fmt.Sprintf("release/v1/providers/%s/%s/versions", namespace, provider)

	shaSumContents, err := getShaSumContents(distPath, repoName, version)
	if err != nil {
		return err
	}

	// Build the versions file...
	platforms := ""
	for _, line := range shaSumContents {
		fileName := line[1] // zip file name

		// get os and arch from filename
		removeFileExtension := strings.Split(fileName, ".zip")
		fileNameSplit := strings.Split(removeFileExtension[0], "_")

		// Get build target and architecture from the zip file name
		target := fileNameSplit[2]
		arch := fileNameSplit[3]

		platforms += "{"
    platforms += fmt.Sprintf(`"os": "%s",`, target)
    platforms += fmt.Sprintf(`"arch": "%s"`, arch)
		platforms += "}"
		platforms += ","
	}
	platforms = strings.TrimRight(platforms, ",") // remove trailing comma, json does not allow

	var versions = []byte(fmt.Sprintf(`
{
  "versions": [
    {
      "version": "%s",
      "protocols": [
        "4.0",
        "5.1"
      ],
      "platform": [
        %s
      ]
    }
  ]
}
`, version, platforms))

	writeFile(versionPath, versions)

	return nil
}

func copyShaFiles(destPath, srcPath, repoName, version string) {
	log.Printf("* Copying SHA files in %s directory", srcPath)

	// Copy files from srcPath 
	shaSum := repoName + "_" + version + "_SHA256SUMS"
	shaSumPath := srcPath + "/" + shaSum

	// _SHA256SUMS file
	_, err := copyFile(shaSumPath, destPath + shaSum)
	if err != nil {
		log.Println(err)
	}

	// _SHA256SUMS.sig file
	_, err = copyFile(shaSumPath + ".sig", destPath + shaSum + ".sig")
	if err != nil {
		log.Println(err)
	}
}

func createDownloadsDir(destPath string) (*string, error) {
	log.Printf("* Creating download/ in %s directory", destPath)

	downloadPath := destPath + "download/"
	
	err := createDir(downloadPath)
	if err != nil {
		return nil, err
	}

	return &downloadPath, nil
}

func createTargetDirs(destPath string) (error) {
	log.Printf("* Creating target dirs in %s directory", destPath)

	targets := [4]string{"darwin", "freebsd", "linux", "windows"}

	for _, v := range targets {
		err := createDir(destPath + v)
		if err != nil {
			return err
		}
	}

	return nil
}

func copyBuildZips(destPath, distPath, repoName, version string) (error) {
	log.Println("* Copying build zips")

	shaSumContents, err := getShaSumContents(distPath, repoName, version)
	if err != nil {
		return err
	}

	// Loop through and copy each
	for _, v := range shaSumContents {
		zipName := v[1]
		zipSrcPath := distPath + zipName
		zipDestPath := destPath + zipName

		log.Printf("  - Zip Source: %s", zipSrcPath)
		log.Printf("   - Zip Dest:  %s", zipDestPath)
		
		// Copy the zip
		_, err := copyFile(zipSrcPath, zipDestPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func getShaSumContents(distPath, repoName, version string) ([][]string, error) {
	shaSumFileName := repoName + "_" + version + "_SHA256SUMS"
	shaSumPath := distPath + "/" + shaSumFileName

	shaSumLine, err := readFile(shaSumPath)
	if err != nil {
		return nil, err
	}

	buildsAndShaSums := [][]string{}

	for _, line := range shaSumLine {
		lineSplit := strings.Split(line, "  ")

		row := []string{lineSplit[0], lineSplit[1]}
		buildsAndShaSums = append(buildsAndShaSums, row)
	}

	// log.Println(buildsAndShaSums)

	return buildsAndShaSums, nil
}

// Create architecture files for each build target
func createArchitectureFiles(namespace, provider, distPath, repoName, version, gpgFingerprint, gpgPubKeyFile, domain string) (error) {
	log.Println("* Creating architecure files in target directories")

	// filename = terraform-provider-[provider]_0.0.1_darwin_amd64.zip - provider_name + version + target + architecture + .zip
	prefix := fmt.Sprintf("v1/providers/%s/%s/%s/", namespace, provider, version)
	pathPrefix := fmt.Sprintf("release/%s", prefix)
	urlPrefix := fmt.Sprintf("https://%s/%s", domain, prefix)

	// download url = https://example.com/v1/providers/namespace/provider/0.0.1/download/terraform-provider_0.0.1_darwin_amd64.zip
	downloadUrlPrefix := urlPrefix + "download/"
	downloadPathPrefix := pathPrefix + "download/"

	// shasums url = https://example.com/v1/providers/namespace/provider/0.0.1/terraform-provider_0.0.1_SHA256SUMS
	shasumsUrl := urlPrefix + fmt.Sprintf("%s_%s_SHA256SUMS", repoName, version)
	// shasums_signature_url = https://example.com/v1/providers/namespace/provider/0.0.1/terraform-provider_0.0.1_SHA256SUMS.sig
	shasumsSigUrl := shasumsUrl + ".sig"

	shaSumContents, err := getShaSumContents(distPath, repoName, version)
	if err != nil {
		return err
	}

	// Get contents of GPG key
	gpgFile, err := readFile(gpgPubKeyFile)
	if err != nil {
		log.Printf("Error reading '%s' file: %s", gpgPubKeyFile, err)
	}

	// loop through every line and stick with \\n
	gpgAsciiPub := ""
	for _, line := range gpgFile {
		gpgAsciiPub = gpgAsciiPub + line + "\\n"
	}
	// log.Println(gpgAsciiPub)

	for _, line := range shaSumContents {
		shasum := line[0] // shasum of the zip
		fileName := line[1] // zip file name

		downloadUrl := downloadUrlPrefix + fileName

		// get os and arch from filename
		removeFileExtension := strings.Split(fileName, ".zip")
		fileNameSplit := strings.Split(removeFileExtension[0], "_")

		// Get build target and architecture from the zip file name
		target := fileNameSplit[2]
		arch := fileNameSplit[3]
		
		// build filepath
		archFileName := downloadPathPrefix + target + "/" + arch 

		var architectureTemplate = []byte(fmt.Sprintf(`
{
  "protocols": [
    "4.0",
    "5.1"
  ],
  "os": "%s",
  "arch": "%s",
  "filename": "%s",
  "download_url": "%s",
  "shasums_url": "%s",
  "shasums_signature_url": "%s",
  "shasum": "%s",
  "signing_keys": {
    "gpg_public_keys": [
      {
        "key_id": "%s",
        "ascii_armor": "%s",
        "trust_signature": "",
        "source": "",
        "source_url": ""
      }
    ]
  }
}
`, target, arch, fileName, downloadUrl, shasumsUrl, shasumsSigUrl, shasum, gpgFingerprint, gpgAsciiPub))

		log.Printf("  - Arch file: %s", archFileName)

		err := writeFile(archFileName, architectureTemplate)
		if err != nil {
			return err
		}
	}

	return nil
}

func createDir(path string) (error) {
	err := os.Mkdir(path, os.ModePerm)
	return err
}

func copyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func readFile(filePath string) ([]string, error) {
	readFile, err := os.Open(filePath)

	if err != nil {
		return nil, err
	}
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	var fileLines []string

	for fileScanner.Scan() {
		fileLines = append(fileLines, fileScanner.Text())
	}

	readFile.Close()

	return fileLines, nil
}

func writeFile(fileName string, fileContents []byte) (error) {
	err := os.WriteFile(fileName, fileContents, 0644)
	return err
}
