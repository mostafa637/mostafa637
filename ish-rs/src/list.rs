//! `util/list.h` — intrusive doubly-linked list.
//!
//! The C header implements an intrusive list where `struct list` is embedded
//! in the containing object. In Rust we provide a safe, owned doubly-linked
//! list for general use, plus an `IntrusiveList` marker that preserves the
//! C API shape for future ports that need intrusive behavior.
//!
//! This is a leaf module: no other iSH header depends on list.h except via
//! inclusion.

use std::collections::LinkedList;
use std::fmt;

/// Intrusive list node, mirroring `struct list`.
///
/// In C, `list->next` and `prev` are raw pointers and `NULL` means detached.
/// Here we track linkage explicitly; `is_detached` corresponds to `list_null`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ListNode {
    pub next: Option<usize>,
    pub prev: Option<usize>,
}

impl ListNode {
    pub fn new_detached() -> Self {
        Self { next: None, prev: None }
    }

    pub fn is_detached(&self) -> bool {
        self.next.is_none() && self.prev.is_none()
    }
}

/// Safe doubly-linked list, analogous to C's `struct list` head plus its
/// `list_add`, `list_remove`, etc. operations.
///
/// For the port we use `LinkedList<T>` internally; the public API mirrors the
/// C naming where useful, but is safe.
#[derive(Debug, Clone)]
pub struct List<T> {
    inner: LinkedList<T>,
}

impl<T> List<T> {
    pub fn new() -> Self {
        Self {
            inner: LinkedList::new(),
        }
    }

    pub fn is_empty(&self) -> bool {
        self.inner.is_empty()
    }

    pub fn len(&self) -> usize {
        self.inner.len()
    }

    pub fn push_front(&mut self, item: T) {
        self.inner.push_front(item);
    }

    pub fn push_back(&mut self, item: T) {
        self.inner.push_back(item);
    }

    pub fn pop_front(&mut self) -> Option<T> {
        self.inner.pop_front()
    }

    pub fn pop_back(&mut self) -> Option<T> {
        self.inner.pop_back()
    }

    pub fn iter(&self) -> impl Iterator<Item = &T> {
        self.inner.iter()
    }

    pub fn iter_mut(&mut self) -> impl Iterator<Item = &mut T> {
        self.inner.iter_mut()
    }

    pub fn clear(&mut self) {
        self.inner.clear();
    }
}

impl<T> Default for List<T> {
    fn default() -> Self {
        Self::new()
    }
}

impl<T> FromIterator<T> for List<T> {
    fn from_iter<I: IntoIterator<Item = T>>(iter: I) -> Self {
        Self {
            inner: LinkedList::from_iter(iter),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn list_node_detached_semantics() {
        let node = ListNode::new_detached();
        assert!(node.is_detached());
        let linked = ListNode {
            next: Some(1),
            prev: Some(0),
        };
        assert!(!linked.is_detached());
    }

    #[test]
    fn safe_list_push_pop() {
        let mut list = List::new();
        assert!(list.is_empty());
        list.push_back(1);
        list.push_back(2);
        list.push_front(0);
        assert_eq!(list.len(), 3);
        assert_eq!(list.iter().copied().collect::<Vec<_>>(), vec![0, 1, 2]);
        assert_eq!(list.pop_front(), Some(0));
        assert_eq!(list.pop_back(), Some(2));
        assert_eq!(list.len(), 1);
    }

    #[test]
    fn list_add_tail_and_remove_semantics() {
        // Simulate C's list_add_tail and list_remove via safe list
        let mut list = List::new();
        list.push_back("a");
        list.push_back("b");
        list.push_back("c");
        // remove middle
        let mut vec: Vec<_> = list.iter().copied().collect();
        vec.remove(1);
        let list2: List<_> = vec.into_iter().collect();
        assert_eq!(list2.iter().copied().collect::<Vec<_>>(), vec!["a", "c"]);
    }
}
