package web

import (
	"fmt"
	"strings"
)

// 用来支持对路由树的操作
// 代表路由树（森林）
type router struct {
	// Beego Gin HTTP method 对应一棵树
	// GET 有一棵树，POST 也有一棵树

	// http method => 路由树根节点
	trees map[string]*node
}

func newRouter() router {
	return router{
		trees: map[string]*node{},
	}
}

// 加一些限制：
// path 必须以 / 开头，不能以 / 结尾，中间也不能有连续的 //
func (r *router) addRoute(method string, path string, handleFunc HandleFunc) {
	if path == "" {
		panic("web: 路径不能为空字符串")
	}
	// 首先找到树来
	root, ok := r.trees[method]

	if !ok {
		// 说明还没有根节点
		root = &node{
			path: "/",
		}
		r.trees[method] = root
	}

	// 开头不能没有/
	if path[0] != '/' {
		panic("web: 路径必须以 / 开头")
	}

	// 结尾
	if path != "/" && path[len(path)-1] == '/' {
		panic("web: 路径不能以 / 结尾")
	}

	// 中间连续 //，可以用 strings.contains("//")

	// 根节点特殊处理一下
	if path == "/" {
		// 根节点重复注册
		if root.handler != nil {
			panic("web: 路由冲突，重复注册[/]")
		}
		root.handler = handleFunc
		return
	}

	// /user/home 被切割成三段
	// 切割这个 path
	segs := strings.Split(path[1:], "/")
	for _, seg := range segs {
		if seg == "" {
			panic("web: 不能有连续的 /")
		}
		// 递归下去，找准位置
		// 如果中途有节点不存在，你就要创建出来
		child := root.childOrCreate(seg)
		root = child
	}
	if root.handler != nil {
		panic(fmt.Sprintf("web: 路由冲突，重复注册[%s]", path))
	}
	root.handler = handleFunc
}

func (r *router) findRoute(method string, path string) (*node, bool) {
	// 基本上是不是也是沿着树深度查找下去？
	root, ok := r.trees[method]
	if !ok {
		return nil, false
	}

	if path == "/" {
		return root, true
	}

	// 这里把前置和后置的 / 都去掉
	path = strings.Trim(path, "/")

	// 按照斜杠切割
	segs := strings.Split(path, "/")
	for _, seg := range segs {
		child, found := root.childOf(seg)
		if !found {
			return nil, false
		}
		root = child
	}
	// 代表我确实有这个节点
	// 但是节点是不是用户注册的有 handler 的，就不一定了
	return root, true

	// return root, root.handler != nil
}

func (n *node) childOrCreate(seg string) *node {
	if seg == "*" {
		n.starChild = &node{
			path: seg,
		}
		return n.starChild
	}
	if n.children == nil {
		n.children = map[string]*node{}
	}
	res, ok := n.children[seg]
	if !ok {
		// 要新建一个
		res = &node{
			path: seg,
		}
		n.children[seg] = res
	}
	return res
}

// childOf 优先考虑静态匹配，匹配不上，再考虑通配符匹配
func (n *node) childOf(path string) (*node, bool) {
	if n.children == nil {
		return n.starChild, n.starChild != nil
	}
	child, ok := n.children[path]
	if !ok {
		return n.starChild, n.starChild != nil
	}
	return child, ok
}

// type tree struct {
// 	root *node
// }

type node struct {
	path string

	// 静态匹配的节点
	// 子 path 到子节点的映射
	children map[string]*node

	// 加一个通配符匹配
	starChild *node

	// 缺一个代表用户注册的业务逻辑
	handler HandleFunc
}
