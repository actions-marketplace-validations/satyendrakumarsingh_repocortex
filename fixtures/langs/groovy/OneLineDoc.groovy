package com.acme.flat

class Flat implements Marker {
    protected final Map<String, Item> items = new LinkedHashMap<String, Item>()
    /** a one-line doc comment: the grammar never closes it */
    int size

    Flat(int size) {
        this.size = size
    }

    void lost() {}
}
