package syncworker

import (
	"context"
	"fmt"
	"time"

	"github.com/anshsaluja/finlink/internal/financialdata"
	"github.com/anshsaluja/finlink/internal/provider"
	"github.com/anshsaluja/finlink/internal/syncs"
)

type ProviderSyncer interface { FetchSyncPage(context.Context,string,string)(provider.SyncResult,error); MarkSynced(context.Context,string,string,time.Time) error }
type DataStore interface { Store(context.Context,[]financialdata.Account,[]financialdata.Transaction) error }
type RunStore interface { MarkStarted(context.Context,int64,time.Time) error; MarkCompleted(context.Context,int64,string,int,int,time.Time) error; MarkFailed(context.Context,int64,string,time.Time) error }
type Worker struct{ providers ProviderSyncer; data DataStore; runs RunStore; now func()time.Time }
func New(providers ProviderSyncer,data DataStore,runs RunStore)*Worker{return &Worker{providers:providers,data:data,runs:runs,now:time.Now}}

// Process executes one durable sync request. The connection cursor is advanced only
// after canonical account/transaction persistence succeeds, making message replay safe.
func(w *Worker)Process(ctx context.Context,event syncs.RequestedEvent) error{
	started:=w.now().UTC(); if err:=w.runs.MarkStarted(ctx,event.RunID,started);err!=nil{return fmt.Errorf("mark sync started: %w",err)}
	result,err:=w.providers.FetchSyncPage(ctx,event.ConnectionID,event.Cursor);if err!=nil{return w.fail(ctx,event.RunID,"provider sync",err)}
	accounts,transactions:=financialdata.Normalize(event.ConnectionID,result,w.now())
	if err:=w.data.Store(ctx,accounts,transactions);err!=nil{return w.fail(ctx,event.RunID,"persist canonical data",err)}
	completed:=w.now().UTC()
	if err:=w.providers.MarkSynced(ctx,event.ConnectionID,result.NextCursor,completed);err!=nil{return w.fail(ctx,event.RunID,"advance connection cursor",err)}
	if err:=w.runs.MarkCompleted(ctx,event.RunID,result.NextCursor,len(accounts),len(transactions),completed);err!=nil{return fmt.Errorf("mark sync completed: %w",err)}
	return nil
}
func(w *Worker)fail(ctx context.Context,runID int64,stage string,cause error)error{wrapped:=fmt.Errorf("%s: %w",stage,cause);if err:=w.runs.MarkFailed(ctx,runID,wrapped.Error(),w.now().UTC());err!=nil{return fmt.Errorf("%v; mark failed: %w",wrapped,err)};return wrapped}
