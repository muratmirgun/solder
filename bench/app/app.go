// Package app is a framework-free service graph used to compare solder and
// fx: 12 components, 3 with Start/Stop hooks.
package app

import "context"

type Config struct{ Addr, DSN string }

type Logger struct{ prefix string }

func NewLogger(c Config) *Logger { return &Logger{prefix: c.Addr} }

type DB struct{ dsn string }

func NewDB(c Config, l *Logger) (*DB, error) { return &DB{dsn: c.DSN}, nil }
func (*DB) Start(context.Context) error      { return nil }
func (*DB) Stop(context.Context) error       { return nil }

type Cache struct{ l *Logger }

func NewCache(l *Logger) *Cache            { return &Cache{l} }
func (*Cache) Start(context.Context) error { return nil }
func (*Cache) Stop(context.Context) error  { return nil }

type UserRepo struct{ db *DB }
type OrderRepo struct{ db *DB }
type ProductRepo struct {
	db    *DB
	cache *Cache
}

func NewUserRepo(db *DB) *UserRepo                 { return &UserRepo{db} }
func NewOrderRepo(db *DB) *OrderRepo               { return &OrderRepo{db} }
func NewProductRepo(db *DB, c *Cache) *ProductRepo { return &ProductRepo{db, c} }

type UserService struct {
	r *UserRepo
	l *Logger
}
type OrderService struct {
	r     *OrderRepo
	users *UserService
	l     *Logger
}
type ProductService struct {
	r *ProductRepo
	l *Logger
}

func NewUserService(r *UserRepo, l *Logger) *UserService { return &UserService{r, l} }
func NewOrderService(r *OrderRepo, u *UserService, l *Logger) *OrderService {
	return &OrderService{r, u, l}
}
func NewProductService(r *ProductRepo, l *Logger) *ProductService { return &ProductService{r, l} }

type Handler struct {
	u *UserService
	o *OrderService
	p *ProductService
}

func NewHandler(u *UserService, o *OrderService, p *ProductService) *Handler {
	return &Handler{u, o, p}
}

type Server struct {
	h    *Handler
	addr string
}

func NewServer(h *Handler, c Config, l *Logger) *Server { return &Server{h, c.Addr} }
func (*Server) Start(context.Context) error             { return nil }
func (*Server) Stop(context.Context) error              { return nil }
