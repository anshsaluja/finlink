package httpserver

import (
 "context"
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 "github.com/anshsaluja/finlink/internal/connections"
 "github.com/anshsaluja/finlink/internal/financialdata"
 "github.com/anshsaluja/finlink/internal/provider"
 "github.com/anshsaluja/finlink/internal/syncs"
)

type paginatedReader struct {
 limits []int
 offsets []int
}
func (p *paginatedReader) AccountsByConnection(context.Context,string)([]financialdata.Account,error){return nil,nil}
func (p *paginatedReader) TransactionsByConnection(_ context.Context,_ string,limit int)([]financialdata.Transaction,error){p.limits=append(p.limits,limit);return []financialdata.Transaction{},nil}
func (p *paginatedReader) TransactionsByConnectionPage(_ context.Context,_ string,limit,offset int)([]financialdata.Transaction,error){
 p.limits=append(p.limits,limit);p.offsets=append(p.offsets,offset)
 return []financialdata.Transaction{},nil
}
type legacyReader struct{}
func (legacyReader) AccountsByConnection(context.Context,string)([]financialdata.Account,error){return nil,nil}
func (legacyReader) TransactionsByConnection(context.Context,string,int)([]financialdata.Transaction,error){return []financialdata.Transaction{},nil}

func transactionHandler(t *testing.T, reader financialDataReader)(http.Handler,string){
 t.Helper()
 registry:=provider.NewRegistry(provider.MockNorthstar{},provider.MockHarbor{})
 cs:=connections.NewService(registry)
 ss:=syncs.NewService(syncs.NewMemoryRepository(),cs,&syncs.MemoryPublisher{})
 handler:=NewWithServicesAndData(cs,ss,reader)
 req:=httptest.NewRequest(http.MethodPost,"/v1/connections",strings.NewReader(`{"provider":"northstar","credentials":{"username":"demo","password":"demo"}}`))
 res:=httptest.NewRecorder();handler.ServeHTTP(res,req)
 if res.Code!=http.StatusCreated{t.Fatalf("connection create: %d %s",res.Code,res.Body.String())}
 var created struct{ID string `json:"id"`}
 if err:=json.Unmarshal(res.Body.Bytes(),&created);err!=nil||created.ID==""{t.Fatalf("invalid connection: %v %s",err,res.Body.String())}
 return handler,"/v1/connections/"+created.ID+"/transactions"
}
func TestTransactionPaginationValidatesBounds(t *testing.T){
 reader:=&paginatedReader{}
 handler,path:=transactionHandler(t,reader)
 for _,tc:=range []struct{query string;status int}{
  {"?limit=200&offset=400",http.StatusOK},
  {"?limit=1&offset=0",http.StatusOK},
  {"?limit=0",http.StatusBadRequest},
  {"?limit=201",http.StatusBadRequest},
  {"?limit=garbage",http.StatusBadRequest},
  {"?offset=-1",http.StatusBadRequest},
  {"?offset=1000001",http.StatusBadRequest},
  {"?offset=garbage",http.StatusBadRequest},
 }{
  t.Run(tc.query,func(t *testing.T){
   res:=httptest.NewRecorder();handler.ServeHTTP(res,httptest.NewRequest(http.MethodGet,path+tc.query,nil))
   if res.Code!=tc.status{t.Fatalf("status=%d want=%d body=%s",res.Code,tc.status,res.Body.String())}
   if tc.query=="?limit=200&offset=400"{
    var body struct{Limit int `json:"limit"`;Offset int `json:"offset"`;MayHaveMore bool `json:"may_have_more"`}
    if err:=json.Unmarshal(res.Body.Bytes(),&body);err!=nil||body.Limit!=200||body.Offset!=400||body.MayHaveMore{t.Fatalf("unexpected page: %+v error=%v",body,err)}
   }
  })
 }
 if len(reader.offsets)!=2||reader.offsets[0]!=400||reader.offsets[1]!=0{t.Fatalf("unexpected offsets: %v",reader.offsets)}
}
func TestTransactionPaginationRejectsUnsupportedLegacyOffset(t *testing.T){
 handler,path:=transactionHandler(t,legacyReader{})
 res:=httptest.NewRecorder();handler.ServeHTTP(res,httptest.NewRequest(http.MethodGet,path+"?offset=1",nil))
 if res.Code!=http.StatusServiceUnavailable{t.Fatalf("status=%d body=%s",res.Code,res.Body.String())}
}
