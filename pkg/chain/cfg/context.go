package cfg

import "context"

type ContextHolder struct {
	ctx context.Context
}

func NewContextHolder() *ContextHolder {
	return &ContextHolder{
		ctx: context.Background(),
	}
}

func (c *ContextHolder) Context() context.Context {
	return c.ctx
}

func (c *ContextHolder) SetContext(ctx context.Context) {
	c.ctx = ctx
}
