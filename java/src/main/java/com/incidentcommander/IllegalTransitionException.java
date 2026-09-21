package com.incidentcommander;

/** Raised when a caller asks for a move the machine forbids. */
public final class IllegalTransitionException extends RuntimeException {
    private static final long serialVersionUID = 1L;

    private final String from;
    private final String to;

    /**
     * Create the exception for a rejected {@code from -> to} move.
     *
     * @param from the state the move started from
     * @param to the rejected destination state
     */
    public IllegalTransitionException(String from, String to) {
        super("illegal transition " + from + " -> " + to);
        this.from = from;
        this.to = to;
    }

    /**
     * The state the move started from.
     *
     * @return the source state
     */
    public String getFrom() {
        return from;
    }

    /**
     * The rejected destination state.
     *
     * @return the destination state
     */
    public String getTo() {
        return to;
    }
}
