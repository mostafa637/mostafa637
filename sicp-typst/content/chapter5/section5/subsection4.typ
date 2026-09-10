// Generated from the SICP XML sources by tools/convert.py — do not edit.
#import "../../../lib/sicp.typ": *

#subsection([Combining Instruction Sequences], label-name: <sec:combining-instruction-sequences>)

#idx("instruction sequence")

This section describes the details on how instruction sequences are
represented and combined. Recall from
section @sec:instruction-sequences that an instruction
sequence is represented as a list of the registers needed, the registers
modified, and the actual instructions. We will also consider a label
(string)
to be a degenerate case of an instruction sequence, which
doesn't need or modify any registers.
So to determine the registers needed
and modified by instruction sequences we use the selectors
#idx("registersneeded", decl: true)#idx("registersmodified", decl: true)#idx("instructions", decl: true)
#snippet(```python
function registers_needed(s) {
    return is_string(s) ? null : head(s);
}
function registers_modified(s) {
    return is_string(s) ? null : head(tail(s));
}
function instructions(s) {
    return is_string(s) ? list(s) : head(tail(tail(s)));
}
```)

and to determine whether a given
sequence needs or modifies a given register we use the predicates
#idx("needsregister", decl: true)#idx("modifiesregister", decl: true)
#snippet(```python
function needs_register(seq, reg) {
    return ! is_null(member(reg, registers_needed(seq)));
}
function modifies_register(seq, reg) {
    return ! is_null(member(reg, registers_modified(seq)));
}
```)

In terms of these predicates and selectors, we can implement the
various instruction sequence combiners used throughout the compiler.

The basic combiner is
#py("append_instruction_sequences").
This takes as
arguments
two
instruction sequences that are to be
executed sequentially and returns an instruction sequence whose statements
are the statements of
the two
sequences appended together.
The subtle point is to determine the registers that are needed and modified by the resulting sequence.
It modifies those registers that
are modified by either sequence;
it needs those registers that must be initialized before the
first sequence can be run (the registers needed by the first sequence), together with those registers needed by
the second sequence that are not initialized (modified) by the first sequence.

The function #py("append_instruction_sequences")
is given two instruction sequences #py("seq1") and
#py("seq2") and returns the instruction sequence whose
instructions
are the
instructions
of #py("seq1") followed
by the
instructions
of #py("seq2"), whose modified
registers are those registers that are modified by either
#py("seq1") or #py("seq2"), and
whose needed registers are the registers needed by
#py("seq1") together with those registers needed by
#py("seq2") that are not modified by
#py("seq1"). (In terms of set operations, the new set
of needed registers is the union of the set of registers needed by
#py("seq1") with the set difference of the registers
needed by #py("seq2") and the registers modified by
#py("seq1").) Thus,
#py("append_instruction_sequences")
is implemented as follows:
#idx("appendinstructionsequences", decl: true)
#snippet(```python
function append_instruction_sequences(seq1, seq2) {
    return make_instruction_sequence(
               list_union(registers_needed(seq1),
                          list_difference(registers_needed(seq2),
                                          registers_modified(seq1))),
               list_union(registers_modified(seq1),
                          registers_modified(seq2)),
               append(instructions(seq1), instructions(seq2)));
}
```)

This
function
uses some simple operations for manipulating sets
represented as lists, similar to the (unordered) set representation
described in section @sec:representing-sets:
#idx("listunion", decl: true)#idx("listdifference", decl: true)
#snippet(```python
function list_union(s1, s2) {
    return is_null(s1)
           ? s2
           : is_null(member(head(s1), s2))
           ? pair(head(s1), list_union(tail(s1), s2))
           : list_union(tail(s1), s2);
}
function list_difference(s1, s2) {
    return is_null(s1)
           ? null
           : is_null(member(head(s1), s2))
           ? pair(head(s1), list_difference(tail(s1), s2))
           : list_difference(tail(s1), s2);
}
```)

The function #py("preserving"),
the second major instruction
sequence combiner, takes a list of registers
#py("regs") and two instruction sequences
#py("seq1") and #py("seq2") that
are to be executed sequentially. It returns an instruction sequence whose
instructions
are the
instructions
of #py("seq1") followed
by the
instructions
of #py("seq2"), with appropriate
#py("save") and #py("restore")
instructions around #py("seq1") to protect the
registers in #py("regs") that are modified by
#py("seq1") but needed by
#py("seq2"). To accomplish this,
#py("preserving") first creates a sequence that has
the required #py("save")s followed by the
instructions
of #py("seq1") followed by the required
#py("restore")s. This sequence needs the registers
being saved and restored in addition to the registers needed by
#py("seq1"), and modifies the registers modified by
#py("seq1") except for the ones being saved and
restored. This augmented sequence and #py("seq2")
are then appended in the usual way. The following
function
implements this strategy recursively, walking down the list of registers to
be preserved:
#idx("preserving", decl: true)
#snippet(```python
function preserving(regs, seq1, seq2) {
    if (is_null(regs)) {
        return append_instruction_sequences(seq1, seq2);
    } else {
        const first_reg = head(regs);
        return needs_register(seq2, first_reg) &&
               modifies_register(seq1, first_reg)
               ? preserving(tail(regs),
                     make_instruction_sequence(
                         list_union(list(first_reg),
                                    registers_needed(seq1)),
                         list_difference(registers_modified(seq1),
                                         list(first_reg)),
                         append(list(save(first_reg)),
                                append(instructions(seq1),
                                       list(restore(first_reg))))),
                     seq2)
               : preserving(tail(regs), seq1, seq2);
    }
}
```)

Another sequence combiner,
#py("tack_on_instruction_sequence"),
is used by
#py("compile_lambda_expression")
to append a
function
body to another sequence. Because the
function
body is not "in line" to be executed as part of the combined
sequence, its register use has no impact on the register use of the sequence
in which it is embedded. We thus ignore the
function
body's sets of needed and modified
registers when we tack it onto the other sequence.
#idx("tackoninstructionsequence", decl: true)
#snippet(```python
function tack_on_instruction_sequence(seq, body_seq) {
    return make_instruction_sequence(
               registers_needed(seq),
               registers_modified(seq),
               append(instructions(seq), instructions(body_seq)));
}
```)

The functions #py("compile_conditional")
and
#py("compile_function_call")
use a special combiner called
#py("parallel_instruction_sequences")
to append the two alternative branches that follow a test. The two branches
will never be executed sequentially; for any particular evaluation of the
test, one branch or the other will be entered. Because of this, the
registers needed by the second branch are still needed by the combined
sequence, even if these are modified by the first branch.
#idx("parallelinstructionsequences", decl: true)
#snippet(```python
function parallel_instruction_sequences(seq1, seq2) {
    return make_instruction_sequence(
               list_union(registers_needed(seq1),
                          registers_needed(seq2)),
               list_union(registers_modified(seq1),
                          registers_modified(seq2)),
               append(instructions(seq1), instructions(seq2)));
}
```)

#idx("instruction sequence")
