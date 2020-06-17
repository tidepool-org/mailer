package worker

import (
	"context"
	"encoding/json"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/tidepool-org/mailer/mailer"
	"go.uber.org/zap"
	"sync"

	pb "github.com/tidepool-org/workscheduler/workscheduler"
	"google.golang.org/grpc"
)

type Worker struct {
	client               pb.WorkSchedulerClient
	logger               *zap.SugaredLogger
	mailerr              mailer.Mailer
	workschedulerAddress string
}

type Params struct {
	Logger               *zap.SugaredLogger
	Mailerr              mailer.Mailer
	WorkschedulerAddress string
}

func New(params Params) *Worker {
	return &Worker{
		client:               nil,
		logger:               params.Logger,
		mailerr:              params.Mailerr,
		workschedulerAddress: params.WorkschedulerAddress,
	}
}

func (w *Worker) Start(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	conn, err := grpc.Dial(w.workschedulerAddress, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return errors.Wrap(err, "could not connect to workscheduler")
	}
	defer conn.Close()

	w.logger.Info("Successfully connected to workscheduler")

	w.client = pb.NewWorkSchedulerClient(conn)
	clientCtx, _ := context.WithCancel(ctx)

	w.logger.Info("Starting work poll loop")

pollLoop:
	for {
		select {
		case <-ctx.Done():
			break pollLoop
		default:
			work, err := w.poll(ctx)
			if err != nil {
				continue
			}

			email, err := w.unmarshalOrFail(clientCtx, work)
			if err != nil {
				continue
			}

			err = w.mailerr.Send(clientCtx, email)
			if err != nil {
				_ = w.fail(clientCtx, work)
				continue
			}

			_ = w.complete(clientCtx, work)
		}
	}

	return nil
}

func (w *Worker) poll(ctx context.Context) (*pb.Work, error) {
	work, err := w.client.Poll(ctx, &empty.Empty{})
	if err != nil {
		w.logger.Error(errors.Wrap(err, "error while polling scheduler for work"))
	}
	return work, err
}

func (w *Worker) unmarshalOrFail(ctx context.Context, work *pb.Work) (*mailer.Email, error) {
	email := &mailer.Email{}
	err := json.Unmarshal(work.Data, email)
	if err != nil {
		w.logger.Error("error unmarshaling work to email", "error", err)
		err = w.fail(ctx, work)
	}
	return email, err
}

func (w *Worker) complete(ctx context.Context, work *pb.Work) error {
	_, err := w.client.Complete(ctx, work.Source)
	if err != nil {
		w.logger.Error(errors.Wrap(err, "error notifying workscheduler for completed work"))
	}
	return err
}

func (w *Worker) fail(ctx context.Context, work *pb.Work) error {
	_, err := w.client.Failed(ctx, work.Source)
	if err != nil {
		w.logger.Error(errors.Wrap(err, "error notifying workscheduler for failed work"))
	}
	return err
}