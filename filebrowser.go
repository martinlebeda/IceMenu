package main

import (
    "io/ioutil"
    "log"
    "path/filepath"
    "strings"
    "github.com/kardianos/osext"
    "sort"
    "golang.org/x/text/unicode/norm"
    "golang.org/x/text/transform"
    "unicode"
    "testing"
    "flag"
    "os"
    "regexp"
    "fmt"
)

type MenuItem struct {
    group, label, icon, command, subPipeMenu string
    // List<VOObMenuItem> submenu = new ArrayList<>() // TODO Lebeda -  práce s podmenu
}

// We implement sort.Interface - Len, Less, and Swap - on our type
// https://gobyexample.com/sorting-by-functions
type MenuItemList []MenuItem

func (s MenuItemList) Len() int {
    return len(s)
}
func (s MenuItemList) Swap(i, j int) {
    s[i], s[j] = s[j], s[i]
}
func isMn(r rune) bool {
    return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}
func (s MenuItemList) Less(i, j int) bool {
    return NormalizeString(s[i].label) < NormalizeString(s[j].label)
}

// TODO Lebeda - napsat testy a vyseparovat
// Remove diacritics and make lowercase.
// http://stackoverflow.com/questions/26722450/remove-diacritics-using-go
func NormalizeString(s string) string {
    l := strings.ToLower(s)
    t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
    n, _, _ := transform.String(t, l)
    return n
}
func TestNormalizeString(t *testing.T) {
    const exp string = "priliz zlutoucky kun upel dabelske ody"
    normalizeString := NormalizeString("Příliš žluťoučký kůň úpěl ďábelské ódy")
    if (exp != normalizeString) {
        t.Error("Expected ", exp, ", got ", normalizeString)
    }
}

const C_NOGROUP = "_NOGROUP_"
const C_NOICON = "_NOICON_"
const C_SEPARATOR = "separator"
const C_STATIC_FILENAME = "static"

var file_menu_item_list []MenuItem
var dir_menu_item_list []MenuItem

// cmd options
var baseDirNamePtr *string
var avfsDirNamePtr *string
var annexOpenScriptPtr *string
var openScriptPtr *string
var suffixFilterPtr *string
var norecursivePtr *bool

func main() {
    // initialize some var
    homedir := os.Getenv("HOME")

    // commandline flags
    baseDirNamePtr = flag.String("dir", homedir, "directories for menu, : is separator")
    avfsDirNamePtr = flag.String("avfs", "", "mountpoint of avfsdirectory")
    annexOpenScriptPtr = flag.String("annex", "", "open script for git annex managed files")
    openScriptPtr = flag.String("open", "xdg-open", "open script for files")
    suffixFilterPtr = flag.String("suffix", "", "include only suffixes, default is all, : is separator")
    norecursivePtr = flag.Bool("norecur", false, "prevent recursive directories ")
    flag.Parse()

    dirs := strings.Split(*baseDirNamePtr, ":")

    for index, dirElement := range dirs {
        if index > 0 {
           fmt.Println(C_SEPARATOR)
        }

        // list directories and files
        files, err := ioutil.ReadDir(dirElement)
        if err != nil {
            log.Fatal(err)
        }

        // static items
        globalStaticMenu := printDirMenu(filepath.Join(homedir, ".icewm", "filebrowser"), dirElement, files) // global
        localStaticMenu := printDirMenu(filepath.Join(dirElement, ".filebrowser"), dirElement, files) // local
        parentStaticMenu := printParentStaticMenu(filepath.Join(dirElement, ".filebrowser"), dirElement, files) // parent static Menu

        if globalStaticMenu || localStaticMenu || parentStaticMenu {
            fmt.Println(C_SEPARATOR)
        }


        // define dynamic items by content
        file_menu_item_list = make([]MenuItem, 0)
        dir_menu_item_list = make([]MenuItem, 0)

        for _, file := range files {
            // TODO Lebeda - default - insert param for switch hiden files
            if (!strings.HasPrefix(file.Name(), ".")) {
                file_menu_item_list, dir_menu_item_list = addFileDirMenuItem(file, dirElement, file_menu_item_list, dir_menu_item_list)
            }
        }

        // sort menu items case and diacritics insensitive
        sort.Sort(MenuItemList(dir_menu_item_list))
        sort.Sort(MenuItemList(file_menu_item_list))

        // print list of menuItems
        PrintMenuItems(dir_menu_item_list)
        if (len(dir_menu_item_list) > 0 && len(file_menu_item_list) > 0) {
            fmt.Println(C_SEPARATOR)
        }
        PrintMenuItems(file_menu_item_list)

    }
}

func printParentStaticMenu(configDir string, baseDirName string, files []os.FileInfo) bool {
    result := printDirMenu(filepath.Join(configDir, ".filebrowser_recursive"), baseDirName, files)

    parentDir := filepath.Dir(configDir)
    if len(parentDir) > 1 {
        result = result || printParentStaticMenu(parentDir, baseDirName, files)
    }

    return result;
}

func printDirMenu(configDir string, baseDirName string, files []os.FileInfo) bool {
    staticSeparator := false

    staticMenuPath := filepath.Join(configDir, C_STATIC_FILENAME)
    if _, err := os.Stat(staticMenuPath); err == nil {
        dat, err := ioutil.ReadFile(staticMenuPath)
        CheckErr(err)

        replace := strings.Replace(string(dat), "%s", baseDirName, -1)
        fmt.Println(replace)
        staticSeparator = true;
    }

    // global dynamic items - git, mvn...
    cndFiles, err := ioutil.ReadDir(configDir)
    if err == nil {
        for _, cndFile := range cndFiles {
            name := cndFile.Name()
            cndAbsFilename, err := filepath.Abs(filepath.Join(configDir, name))
            CheckErr(err)

            // skip static menu
            if (name == C_STATIC_FILENAME) {
                continue
            }
            if (checkNameInDirlist(files, name)) {
                dat, err := ioutil.ReadFile(cndAbsFilename)
                CheckErr(err)
                replace := strings.Replace(string(dat), "%s", baseDirName, -1)
                fmt.Println(replace)
                staticSeparator = true;
            }
        }
    }

    return staticSeparator
}

func isAnnexDirectory(dir string) bool {
        parentDir := filepath.Dir(dir)
        annexDir := filepath.Join(dir, ".git", "annex")
        _, err := os.Stat(annexDir);
        if len(parentDir) > 1 && os.IsNotExist(err) {
            return isAnnexDirectory(parentDir)
        } else {
            return err == nil;
        }
}

func addFileDirMenuItem(fileItem os.FileInfo, base_dir_name string, file_menu_item_list []MenuItem, dir_menu_item_list []MenuItem) ([]MenuItem, []MenuItem) {
    absfilename, err := filepath.Abs(filepath.Join(base_dir_name, fileItem.Name()))
    name := fileItem.Name()
    CheckErr(err)

    avfsSuffixes, _ := regexp.Compile(`(?i).*\.(zip|tar\.gz|tgz|tar\.bz2|tgz|7z|rar)$`)

    if (fileItem.Mode() & os.ModeSymlink == os.ModeSymlink) {
        // eval link
        linkTarget, _ := filepath.EvalSymlinks(absfilename)

        // get base path
        var linkBasePath string
        if filepath.IsAbs(linkTarget) {
            linkBasePath = filepath.Dir(linkTarget)
        } else {
            linkBasePath, _ = os.Getwd()
        }

        // runc recursive for target
        fileTaget, err := os.Lstat(linkTarget)
        if err == nil {
            if fileTaget.IsDir() {
                file_menu_item_list, dir_menu_item_list =
                        addFileDirMenuItem(fileTaget, linkBasePath, file_menu_item_list, dir_menu_item_list)
            } else {
                file_menu_item_list = appendFileItem(file_menu_item_list, getFileItem(*annexOpenScriptPtr, name, absfilename))
            }
        } else if len(*annexOpenScriptPtr) > 0 && isAnnexDirectory(base_dir_name) {
            file_menu_item_list = appendFileItem(file_menu_item_list, getFileItem(*annexOpenScriptPtr, name, absfilename))
        }
    } else if avfsSuffixes.MatchString(name) && len(*avfsDirNamePtr) > 0 {
        dir_menu_item_list = append(dir_menu_item_list, getDirItem(name, *avfsDirNamePtr + absfilename + "#"))
        file_menu_item_list = appendFileItem(file_menu_item_list, getFileItem(*openScriptPtr, name, absfilename))
    } else if (!fileItem.IsDir()) {
        file_menu_item_list = appendFileItem(file_menu_item_list, getFileItem(*openScriptPtr, name, absfilename))
    } else {
        if !*norecursivePtr {
            dir_menu_item_list = append(dir_menu_item_list, getDirItem(name, absfilename))
        }
    }

    return file_menu_item_list, dir_menu_item_list
}

func appendFileItem(slice []MenuItem, elementPtr *MenuItem) []MenuItem {
    if elementPtr != nil {
        slice = append(slice, *elementPtr)
    }
    return slice
}

func getFileItem(openCmd string, name string, absFilename string) *MenuItem {
    if isSuffixOk(absFilename) {
        menu_item := MenuItem{C_NOGROUP, name, C_NOICON, "", ""}
        menu_item.command = openCmd + " '" + absFilename + "'"
        menu_item.icon = "file"
        return &menu_item
    } else {
        return nil
    }
}

func isSuffixOk(name string) bool {
   if len(*suffixFilterPtr) > 0	{
       for _, sfx := range strings.Split(*suffixFilterPtr, ":") {
           if strings.HasSuffix(name, sfx) {
               return true
           }
       }
       return false
   } else {
       return true
   }
}

func getDirItem(name string, absFilename string) MenuItem {
    selfname, _ := osext.Executable()

    menu_item := MenuItem{C_NOGROUP, name, C_NOICON, "", ""}
    avfsOpt := ""
    if len(*avfsDirNamePtr) > 0 {
        avfsOpt = " -avfs " + *avfsDirNamePtr
    }
    annexOpt := ""
    if len(*annexOpenScriptPtr) > 0 {
        annexOpt = " -annex " + *annexOpenScriptPtr
    }

    menu_item.subPipeMenu = selfname + avfsOpt + annexOpt + " -dir '" + absFilename + "'"
    dir_menu_item_list = append(dir_menu_item_list, menu_item)

    return menu_item
}

func checkNameInDirlist(files []os.FileInfo, name string) bool {
    pattern := strings.Replace(name, "_", ".*", -1)
    r, _ := regexp.Compile(pattern)
    for _, file := range files {
        if (r.MatchString(file.Name())) {
            return true
        }
    }
    return false
}

// print list of menuitems formated
func PrintMenuItems(menu_item_list []MenuItem) {
    groupOld := C_NOGROUP

    for _, item := range menu_item_list {
        //fmt.Println(item.command)

        if (groupOld != item.group) {
            if (groupOld != C_NOGROUP) {
                fmt.Println(C_SEPARATOR)
            }
            groupOld = item.group
        }
        if (item.command != "") {
            fmt.Println("prog \"" + escapeSpecial(item.label) + "\"    " + getIcon(item.icon) + "    " + item.command)
        }
        if (item.subPipeMenu != "") {
            fmt.Println("menuprogreload \"" + escapeSpecial(item.label) + "\" " + C_NOICON + " 0 " + item.subPipeMenu)
        }

    }
}

// return icon name with default if empty
func getIcon(iconName string) string {
    if (iconName != "") {
        return iconName
    } else {
        return C_NOICON
    }
}

// replace special characters
func escapeSpecial(label string) string {
    result := strings.Replace(label, "_", "__", 1)
    return result
}
//static String escapeSpecial(String s, boolean skipUnderScore = false) {
////        s = s.replaceAll(/&/, "&amp;")
////        s = s.replaceAll(/"/, "&quot;")
////        s = s.replaceAll(/</, "&lt;")
////        s = s.replaceAll(/>/, "&gt;")
//
//        if (!skipUnderScore) {
//            s = s.replaceFirst(/\_/, "__")
////            s = s.replaceAll(/\_/, "__")
//        }
//        return s
//    }

// check error in functions
func CheckErr(err error) {
    if err != nil {
        //log.Fatal(err)
        panic(err)
    }
}