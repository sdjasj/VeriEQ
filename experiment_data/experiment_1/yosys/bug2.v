module a (
    input b,
    input h,
    input i,
    output reg o
);
    wire c, j, k, e, g;

    reg l, n, d, f;

    always @(posedge b or negedge c) begin
        if (c) begin
            o = j > l;
            l = (l | j) ? k ? h : i ? 0 : n : 0;      
        end
    end

    always @(h) begin
        d = k;
    end

    always @(e or g or f) begin
        if (e) begin
            l = 0;
        end else begin
            d = g;
            l = f;
        end
    end

endmodule