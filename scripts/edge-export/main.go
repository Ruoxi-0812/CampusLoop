// Export the existing Go page templates without coupling navigation to backend startup.
package main
import("bytes";"html/template";"os";"strings";"path/filepath")
func main(){
 t:=template.Must(template.New("").Funcs(template.FuncMap{"renderMoney":func(any)string{return ""},"renderCurrencyLogo":func(any)string{return ""}}).ParseGlob("src/frontend/templates/*.html"))
 data:=map[string]any{"baseUrl":"","marketplaceEnabled":true,"show_currency":false,"cart_size":0}
 render:=func(name string)string{var b bytes.Buffer;if err:=t.ExecuteTemplate(&b,name,data);err!=nil{panic(err)};return b.String()}
 write:=func(name,html string){html=strings.ReplaceAll(html,"window.CampusLoopBackendConfig = {baseUrl:","window.CampusLoopBackendConfig = {edge: true, baseUrl:");html=strings.ReplaceAll(html,"<script src=\"/static/js/marketplace.js\">","<script src=\"/edge/connection.js\"></script><script src=\"/static/js/marketplace.js\">");lines:=strings.Split(html,"\n");for i:=range lines{lines[i]=strings.TrimRight(lines[i]," \t")};html=strings.Join(lines,"\n");if err:=os.WriteFile(filepath.Join("vercel-demo/edge",name+".html"),[]byte(html),0644);err!=nil{panic(err)}}
 for _,name:=range []string{"signin","post-item","my-listings","messages"}{write(name,render(name))}
 body,err:=os.ReadFile("scripts/edge-export/product-body.html");if err!=nil{panic(err)}
 product := string(body)
 for _,path := range []string{"static/js/chat.js", "edge/product-data.js", "edge/product-preview.js"} {
  script,err := os.ReadFile(filepath.Join("vercel-demo",path)); if err!=nil{panic(err)}
  product = strings.ReplaceAll(product, `<script src="/`+path+`"></script>`, "<script>"+string(script)+"</script>")
 }
 write("product",render("header")+product+render("footer"))
}
