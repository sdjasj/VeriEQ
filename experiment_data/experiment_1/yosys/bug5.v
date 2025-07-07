module a (
    input  [2:0] b,
    input        e,
    output       c
);
    wire [31:0]  d;
    reg  [31:0]  h;
    reg          f;
    reg          g;
    reg          i;
    
    always @(posedge e) begin
        c <= b;
        h <= b;
    end
    
    always @(*) begin
        e = f;
        case (f)
            'd1: begin
                f = d * h ? 0 : g;
                h = f + i;
            end
        endcase
    end
endmodule