package queue

import (
	"context"
	"github.com/pkg/errors"
	"go-sniffer/pkg/client"
	"go-sniffer/pkg/model"
)

type SnifferBackendResult struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data string `json:"data"`
}

const (
	SnifferBackendPath = "/sqlanalysis/V1.0.0/receivemsg/receiveMsg"
)

func (k *MetricsQueue) PushMetrics(parm *model.MysqlQueryPiece) (err error) {
	cli := client.NewBaseClient(k.Host, 0)
	var result SnifferBackendResult
	err = cli.Post(context.Background(), SnifferBackendPath, &parm, &result)
	if err != nil {
		err = errors.Wrapf(err, "fail to call %s get security user", client.SnifferBackendUpstream)
		return
	}

	if result.Code != "00000" {
		err = errors.Wrapf(errors.New("fail to upload sql metrics"),
			"code: %s,message: %s, raw: %v", result.Code, result.Msg, parm)
		return err
	}

	return
}
