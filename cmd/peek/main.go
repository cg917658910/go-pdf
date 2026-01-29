package main

import (
	"fmt"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func main(){
	if len(os.Args)<2{fmt.Println("usage: peek file"); os.Exit(1)}
	f:=os.Args[1]
	ctx,err:=api.ReadContextFile(f)
	if err!=nil{fmt.Printf("read context %s: %v\n",f,err); os.Exit(1)}
	for i:=1;i<=ctx.PageCount;i++{
		page,_,_,_:=ctx.PageDict(i,true)
		c:=page["Contents"]
		fmt.Printf("page %d Contents type: %T\n",i,c)
		switch t:=c.(type){
		case types.IndirectRef:
			if sd,_,err:=ctx.DereferenceStreamDict(t);err==nil{fmt.Printf(" deref stream dict ok\n"); _=sd} else {fmt.Printf(" deref stream dict err: %v\n",err)}
		case types.LazyObjectStreamObject:
			fmt.Printf(" got LazyObjectStreamObject\n")
		default:
			fmt.Printf(" other: %T\n",t)
		}
	}
}
