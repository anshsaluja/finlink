package main

import(
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"
	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
	"github.com/anshsaluja/finlink/internal/config"
	"github.com/anshsaluja/finlink/internal/connections"
	"github.com/anshsaluja/finlink/internal/financialdata"
	"github.com/anshsaluja/finlink/internal/provider"
	"github.com/anshsaluja/finlink/internal/syncs"
	"github.com/anshsaluja/finlink/internal/syncworker"
)
func main(){cfg,err:=config.Load();if err!=nil{log.Fatal(err)};db,err:=sql.Open("postgres",cfg.DatabaseURL);if err!=nil{log.Fatalf("open postgres: %v",err)};defer db.Close();ctx,cancel:=context.WithTimeout(context.Background(),5*time.Second);if err:=db.PingContext(ctx);err!=nil{cancel();log.Fatalf("ping postgres: %v",err)};cancel()
	registry:=provider.NewRegistry(provider.MockNorthstar{},provider.MockHarbor{});connectionsService:=connections.NewServiceWithRepository(registry,connections.NewSQLRepository(db));runs:=syncs.NewSQLRepository(db);worker:=syncworker.New(connectionsService,financialdata.NewSQLRepository(db),runs)
	reader:=kafka.NewReader(kafka.ReaderConfig{Brokers:cfg.KafkaBrokers,Topic:cfg.KafkaTopic,GroupID:cfg.KafkaConsumerGroup,MinBytes:1,MaxBytes:10e6,MaxWait:time.Second});defer reader.Close();log.Printf("finlink sync worker consuming %s",cfg.KafkaTopic)
	for{message,err:=reader.FetchMessage(context.Background());if err!=nil{log.Fatalf("fetch kafka message: %v",err)};var event syncs.RequestedEvent;if err:=json.Unmarshal(message.Value,&event);err!=nil{log.Printf("invalid sync event at offset %d: %v",message.Offset,err);time.Sleep(time.Second);continue};if err:=worker.Process(context.Background(),event);err!=nil{log.Printf("sync run %d failed: %v",event.RunID,err);time.Sleep(time.Second);continue};if err:=reader.CommitMessages(context.Background(),message);err!=nil{log.Printf("commit kafka message for run %d: %v",event.RunID,err);continue};log.Printf("sync run %d completed",event.RunID)}
}
