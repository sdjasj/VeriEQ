module top (
    output wire [32:1] out1,
    output wire [32'h0fff_ffff:1] out2,
);
    wire signed wire_1 = 1'b1;
    assign out1 = wire_1 >>> 32'h8F7A7A7A;
    assign out2 = wire_1 >>> 36'hfffffff;
endmodule