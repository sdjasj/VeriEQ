module a(b, c, h);
    input wire [22:17] b;
    input wire c;
    output wire h;
    assign h =  32'ha2f % { 1'b1 ? b : c};
endmodule