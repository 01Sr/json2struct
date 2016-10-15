package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconc"

	"strings"
	"time"
)

var (
	Mstring = "string"
	Mbool   = "bool"
	Mint    = "int"
	Mfloat  = "float"
	Marray  = "A$"
	Mobject = "O$"
	result  = ""
	last    time.Time
)

func getType(v string) (string, error) {

	if v == "" {
		return "", errors.New("The value is nil")
	}
	if strings.HasPrefix(v, "\"") {
		return Mstring, nil
	}
	if strings.HasPrefix(v, "A$") {
		return Marray, nil
	}
	if strings.HasPrefix(v, "O$") {
		return Mobject, nil
	}
	if v == "true" || v == "false" {
		return Mbool, nil
	}
	if strings.Contains(v, ".") {
		return Mfloat, nil
	}
	return Mint, nil
}
func checkErr(err error) {
	if err != nil {
		log.Println(err.Error())
	}
}

type Bracket struct {
	LeftP  int        //左下标
	RightP int        //右下表
	B      []*Bracket //嵌套引用
}

func parseJson(s string) {
	s = strings.Replace(s, " ", "", -1)
	s = strings.Replace(s, "\n", "", -1)
	fmt.Println(s)
	s = s[1 : len(s)-1]
	t := makeTree(s)

	generateCode(traverse(s, t), "Root") //生成最外层结构体
}

//映射json或为array的value到Bracket为节点的森林
func makeTree(s string) []*Bracket {
	l := make([]*Bracket, 0, 100)
	flag := false //判断是否生成一个完整的括号索引
	for i, v := range s {
		if v == '{' || v == '[' {
			b := &Bracket{i, -1, nil} //生成节点并加入左标，右标暂为-1
			l = addElemation(l, b)
			continue
		}
		if v == '}' || v == ']' {
			for k := len(l) - 1; k >= 0; k-- { //从后往前遍历切片
				right := l[k].RightP
				if right == -1 { //判断此节点是否已经有右标
					if flag { //判断当前右标是否已经添加至某节点，true将右节点作为子节点加入此节点，并从切片中删除右节点
						if l[k].B == nil {
							l[k].B = make([]*Bracket, 0, 100)
						}
						l[k].B = addElemation(l[k].B, l[k+1])
						l = removeElemation(l, k+1)
						break
					} else { //false添加至此节点
						l[k].RightP = i
						flag = true
					}
				}
			}
		}
		flag = false
	}
	return l
}

func traverse(s string, l []*Bracket) string {
	last := ""
	for _, v := range l {

		if v.B != nil {
			s = traverse(s, v.B)
		}
		re, err := regexp.Compile(`"`)
		if err != nil {
			log.Println(err.Error())
		}
		arr := re.FindAllIndex([]byte(s), -1)
		nL, nR := findSNameP(arr, v.LeftP) //获取与当前value左边界最靠近的两个“"”的位置
		sName := strings.Title(s[nL:nR+1]) + "S"
		if last != sName { //判断是否重复
			last = sName
			generateCode(s[v.LeftP+1:v.RightP], last) //生成结构体
		}

		//替换value为json或者array的项
		left := s[:v.LeftP]
		right := s[v.RightP+1:]
		b := bytes.NewBufferString(Mobject)
		if s[v.LeftP] == '[' {
			b.Reset()
			b.WriteString(Marray)
		}
		len := v.RightP - v.LeftP + 1 - len(b.String())
		for i := 0; i < len; i++ {
			b.WriteString("#")
		}
		s = left + b.String() + right

	}

	return s
}

func findSNameP(arr [][]int, b int) (left int, right int) {
	i := 0
	for i, _ = range arr {
		if b-arr[i][0] < 0 {
			break
		}
	}
	if i == len(arr)-1 {
		i++
	}
	left = arr[i-2][0] + 1
	right = arr[i-1][0] - 1
	return
}

func generateCode(s string, sName string) {
	b := bytes.NewBufferString("type " + sName + " struct{\n")
	arr := strings.Split(s, ",")
	if strings.Contains(arr[0], ":") {
		for _, v := range arr {
			m := strings.Split(v, ":")
			name := strings.TrimFunc(m[0], func(r rune) bool {
				if r == '"' || r == ' ' || r == '\n' {
					return true
				}
				return false
			})
			strconc.Connect(b, "	", strings.Title(name), " ")
			t, err := getType(m[1])
			checkErr(err)
			switch t {
			case Mint:
				b.WriteString("int")
			case Mfloat:
				b.WriteString("float")
			case Mbool:
				b.WriteString("bool")
			case Mstring:
				b.WriteString("string")
			case Mobject:
				b.WriteString(strings.Title(name) + "S")
			case Marray:
				b.WriteString("[]" + strings.Title(name) + "S")
			}
			strconc.Connect(b, " `json:\"", name, "\"`\n")
		}
		b.WriteString("}\n")
	} else {
		b.Reset()
		b.WriteString("type " + sName + " ")
		t, err := getType(arr[0])
		checkErr(err)
		switch t {
		case Mint:
			b.WriteString("int\n")
		case Mfloat:
			b.WriteString("float\n")
		case Mbool:
			b.WriteString("bool\n")
		case Mstring:
			b.WriteString("string\n")
		case Marray:
			b.Reset()
		case Mobject:
			b.Reset()
		default:
			b.WriteString("interface{}\n")
		}
	}
	if !strings.Contains(result, b.String()) {
		result += b.String()
	}

}

func addElemation(l []*Bracket, b *Bracket) []*Bracket {
	length := len(l)
	l = l[:length+1]
	l[length] = b
	return l
}

func removeElemation(l []*Bracket, i int) []*Bracket {
	l = l[:i]
	return l
}

func (this *Bracket) String() string {
	if this == nil {
		return "nil"
	}
	b := bytes.NewBufferString("")
	strconc.Connect(b, "{", this.LeftP, ",", this.RightP, ",")

	if this.B == nil {
		strconc.Connect(b, nil)

	} else {
		strconc.Connect(b, "[")
		for i, v := range this.B {
			strconc.Connect(b, v.String())
			if i != len(this.B)-1 {
				strconc.Connect(b, ",")
			}
		}
		strconc.Connect(b, "]")
	}
	strconc.Connect(b, "}")
	return b.String()

}

func home(w http.ResponseWriter, r *http.Request) {
	if last.IsZero() || time.Now().Sub(last).Seconds() > 0.5 {
		last = time.Now()
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {

		}
		d := string(data)
		if d != "" {

			parseJson(d)
			fmt.Println(result)
			fmt.Fprint(w, result)
			result = ""

		} else {
			fmt.Println("home")
			content, err := ioutil.ReadFile("home.html")
			if err != nil {
				fmt.Fprint(w, "访问错误")
			} else {
				fmt.Fprintf(w, "%s", content)
			}
		}
	}

}

func main() {
	fmt.Println("服务已启动")
	http.HandleFunc("/", home)
	http.ListenAndServe(":7080", nil)
}
