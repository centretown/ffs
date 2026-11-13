package main

import (
	"context"
	"errors"
	"fmt"
)

func main() {
	// mergeCancel returns a context that contains the values of ctx,
	// and which is canceled when either ctx or cancelCtx is canceled.
	mergeCancel := func(ctx, cancelCtx context.Context) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancelCause(ctx)
		stop := context.AfterFunc(cancelCtx, func() {
			cancel(context.Cause(cancelCtx))
		})
		return ctx, func() {
			stop()
			cancel(context.Canceled)
		}
	}

	ctx1, cancel1 := context.WithCancelCause(context.Background())
	defer cancel1(errors.New("ctx1 canceled"))

	ctx2, cancel2 := context.WithCancelCause(context.Background())

	mergedCtx, mergedCancel := mergeCancel(ctx1, ctx2)
	defer func() {
		mergedCancel()
		fmt.Println("ctx1 canceled")
	}()

	cancel2(errors.New("ctx2 canceled"))
	<-mergedCtx.Done()
	fmt.Println(context.Cause(mergedCtx))

}
